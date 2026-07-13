package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/richardtan10176/crypto_analytics/internal/consumer"
)

func main() {
	godotenv.Load()

	brokerAddr := os.Getenv("KAFKA_BROKER")
	topic := os.Getenv("KAFKA_TOPIC")
	dsn := os.Getenv("TIMESCALE_DSN")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal("db:", err)
	}
	defer pool.Close()

	proc := consumer.NewStreamProcessor(func() []consumer.WindowAggregator {
		return []consumer.WindowAggregator{consumer.NewOHLCVAggregator(pool)}
	})

	c := consumer.New(brokerAddr, topic, proc)
	if err := c.Run(ctx); err != nil {
		log.Println("consumer:", err)
	}
	log.Println("consumer stopped")
}
