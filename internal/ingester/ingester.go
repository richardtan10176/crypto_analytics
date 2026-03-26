package ingester

import (
	"context"
	"encoding/json"
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

func (i *Ingester) Run() {
	conn, _, err := websocket.DefaultDialer.Dial(i.wsURL, nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	log.Println("Connected to", i.wsURL)
	for {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("Read error, attempting exponential backoff:", err)
				break
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

			go func(symbol string, payload []byte) {
				if err := i.producer.Publish(context.Background(), symbol, payload); err != nil {
					log.Println("publish error:", err)
				}
			}(raw.Symbol, bytes)
		}

		conn.Close()
		exp := 0
		for {
			conn, _, err = websocket.DefaultDialer.Dial(i.wsURL, nil)
			if err == nil {
				break
			}
			log.Println("Retry failed, waiting...", time.Duration(1<<exp)*time.Second)
			time.Sleep(time.Duration(1<<exp) * time.Second)
			if exp < 5 {
				exp++
			}
		}
	}
}
