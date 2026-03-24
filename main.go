package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
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

func main() {
	godotenv.Load()

	brokerAddr := os.Getenv("KAFKA_BROKER")
	topic := os.Getenv("KAFKA_TOPIC")
	wsURL := os.Getenv("BINANCE_WS_URL")

	p := producer.New(brokerAddr, topic)
	defer p.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	log.Println("Connected to", wsURL)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
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

		if err := p.Publish(context.Background(), raw.Symbol, bytes); err != nil {
			log.Println("publish error:", err)
		}
	}
}
