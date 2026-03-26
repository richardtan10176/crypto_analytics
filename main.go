package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/richardtan10176/crypto_analytics/internal/ingester"
	"github.com/richardtan10176/crypto_analytics/internal/producer"
)

func main() {
	godotenv.Load()

	brokerAddr := os.Getenv("KAFKA_BROKER")
	topic := os.Getenv("KAFKA_TOPIC")
	wsURL := os.Getenv("BINANCE_WS_URL")

	p, err := producer.New(context.Background(), brokerAddr, topic)
	if err != nil {
		log.Fatal("producer:", err)
	}
	defer p.Close()

	i := ingester.New(wsURL, p)
	i.Run()
}
