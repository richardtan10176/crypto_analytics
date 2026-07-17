package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

//go:embed index.html
var indexHTML []byte

type candle struct {
	WindowStart time.Time `json:"t"`
	Open        float64   `json:"o"`
	High        float64   `json:"h"`
	Low         float64   `json:"l"`
	Close       float64   `json:"c"`
	Volume      float64   `json:"v"`
	TradeCount  int64     `json:"n"`
}

type candlesResponse struct {
	Granularity string   `json:"granularity"`
	Candles     []candle `json:"candles"`
}

// granularityFor picks the rollup relation to query based on the requested
// range, so long ranges don't require pulling thousands of 1-minute candles.
// The returned table name is always one of these three literals (never
// derived from request input), so interpolating it into the query is safe.
func granularityFor(hours time.Duration) (table, label string) {
	switch {
	case hours <= 24*time.Hour:
		return "ohlcv_1m", "1m"
	case hours <= 14*24*time.Hour:
		return "ohlcv_1h", "1h"
	default:
		return "ohlcv_1d", "1d"
	}
}

type server struct {
	db *pgxpool.Pool
}

func (s *server) symbols(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `SELECT DISTINCT symbol FROM ohlcv_1m ORDER BY symbol`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	symbols := []string{}
	for rows.Next() {
		var sym string
		if err := rows.Scan(&sym); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		symbols = append(symbols, sym)
	}
	writeJSON(w, symbols)
}

func (s *server) candles(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		http.Error(w, "symbol is required", http.StatusBadRequest)
		return
	}
	hours, err := time.ParseDuration(r.URL.Query().Get("hours") + "h")
	if err != nil || hours < time.Hour || hours > 90*24*time.Hour {
		http.Error(w, "hours must be a number between 1 and 2160", http.StatusBadRequest)
		return
	}

	table, granularity := granularityFor(hours)

	rows, err := s.db.Query(r.Context(), fmt.Sprintf(`
		SELECT window_start, open, high, low, close, volume, trade_count
		FROM %s
		WHERE symbol = $1 AND window_start >= now() - $2::interval
		ORDER BY window_start`, table), symbol, hours.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	candles := []candle{}
	for rows.Next() {
		var c candle
		if err := rows.Scan(&c.WindowStart, &c.Open, &c.High, &c.Low, &c.Close, &c.Volume, &c.TradeCount); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		candles = append(candles, c)
	}
	writeJSON(w, candlesResponse{Granularity: granularity, Candles: candles})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func main() {
	godotenv.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, os.Getenv("TIMESCALE_DSN"))
	if err != nil {
		log.Fatal("db:", err)
	}
	defer pool.Close()

	s := &server{db: pool}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/symbols", s.symbols)
	mux.HandleFunc("GET /api/candles", s.candles)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})

	addr := os.Getenv("DASHBOARD_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	log.Println("dashboard listening on", addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
	log.Println("dashboard stopped")
}
