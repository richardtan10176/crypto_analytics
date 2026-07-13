package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"github.com/richardtan10176/crypto_analytics/internal/ingester"
	"github.com/richardtan10176/crypto_analytics/internal/producer"
)

func main() {
	godotenv.Load()

	brokerAddr := os.Getenv("KAFKA_BROKER")
	topic := os.Getenv("KAFKA_TOPIC")
	wsURL := os.Getenv("BINANCE_WS_URL")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	p, err := producer.New(ctx, brokerAddr, topic)
	if err != nil {
		log.Fatal("producer:", err)
	}
	defer p.Close()

	i := ingester.New(wsURL, p)
	i.Run(ctx)
	log.Println("ingester stopped")
}
