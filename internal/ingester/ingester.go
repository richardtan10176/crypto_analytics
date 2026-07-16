package ingester

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"

	tradepb "github.com/richardtan10176/crypto_analytics/gen/trade"
	"github.com/richardtan10176/crypto_analytics/internal/producer"
)

type rawTrade struct {
	EventType string `json:"e"`
	EventTime int64  `json:"E"`
	Symbol    string `json:"s"`
	TradeID   int64  `json:"t"`
	Price     string `json:"p"`
	Quantity  string `json:"q"`
	TradeTime int64  `json:"T"`
	IsMaker   bool   `json:"m"`
}

type Ingester struct {
	wsURL    string
	producer *producer.Producer
}

func New(wsURL string, p *producer.Producer) *Ingester {
	return &Ingester{
		wsURL:    wsURL,
		producer: p,
	}
}

func (i *Ingester) Run(ctx context.Context) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, i.wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial %s: %w", i.wsURL, err)
	}
	log.Println("Connected to", i.wsURL)

	for {
		i.readLoop(ctx, conn)
		if ctx.Err() != nil {
			return nil
		}

		exp := 0
		for {
			conn, _, err = websocket.DefaultDialer.DialContext(ctx, i.wsURL, nil)
			if err == nil {
				log.Println("Reconnected to", i.wsURL)
				break
			}
			log.Println("Retry failed, waiting...", time.Duration(1<<exp)*time.Second)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Duration(1<<exp) * time.Second):
			}
			if exp < 5 {
				exp++
			}
		}
	}
}

// readLoop reads trades until the connection drops or ctx is cancelled. The
// watcher goroutine closes the connection on cancellation, which is the only
// way to unblock the blocking ReadMessage call.
func (i *Ingester) readLoop(ctx context.Context, conn *websocket.Conn) {
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
		case <-done:
		}
		conn.Close()
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() == nil {
				log.Println("Read error, attempting exponential backoff:", err)
			}
			return
		}

		var raw rawTrade
		if err := json.Unmarshal(msg, &raw); err != nil {
			log.Println("parse error:", err)
			continue
		}

		event := &tradepb.TradeEvent{
			EventType: raw.EventType,
			EventTime: raw.EventTime,
			Symbol:    raw.Symbol,
			TradeId:   raw.TradeID,
			Price:     raw.Price,
			Quantity:  raw.Quantity,
			TradeTime: raw.TradeTime,
			IsMaker:   raw.IsMaker,
		}

		bytes, err := proto.Marshal(event)
		if err != nil {
			log.Println("marshal error:", err)
			continue
		}

		// Publish synchronously: a goroutine per message can reorder trades,
		// which would defeat the per-symbol ordering the message key provides.
		if err := i.producer.Publish(ctx, raw.Symbol, bytes); err != nil {
			log.Println("publish error:", err)
		}
	}
}
