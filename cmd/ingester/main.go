package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/joho/godotenv"

	"github.com/richardtan10176/crypto_analytics/internal/config"
	"github.com/richardtan10176/crypto_analytics/internal/ingester"
	"github.com/richardtan10176/crypto_analytics/internal/producer"
)

func main() {
	godotenv.Load()

	brokerAddr := os.Getenv("KAFKA_BROKER")
	topic := os.Getenv("KAFKA_TOPIC")
	wsBaseURL := os.Getenv("BINANCE_WS_BASE_URL")
	symbols := config.Symbols(os.Getenv("SYMBOLS"))
	if len(symbols) == 0 {
		log.Fatal("SYMBOLS is required (comma-separated, e.g. BTCUSDT,ETHUSDT)")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	p, err := producer.New(ctx, brokerAddr, topic, len(symbols))
	if err != nil {
		log.Fatal("producer:", err)
	}
	defer p.Close()

	var wg sync.WaitGroup
	for _, sym := range symbols {
		wg.Add(1)
		go func(symbol string) {
			defer wg.Done()
			wsURL := fmt.Sprintf("%s/%s@trade", wsBaseURL, strings.ToLower(symbol))
			i := ingester.New(wsURL, p)
			if err := i.Run(ctx); err != nil {
				log.Printf("ingester[%s]: %v", symbol, err)
				return
			}
			log.Printf("ingester[%s] stopped", symbol)
		}(sym)
	}
	wg.Wait()
	log.Println("all ingesters stopped")
}
