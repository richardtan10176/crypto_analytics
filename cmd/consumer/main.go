package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/richardtan10176/crypto_analytics/internal/config"
	"github.com/richardtan10176/crypto_analytics/internal/consumer"
)

func main() {
	godotenv.Load()

	brokerAddr := os.Getenv("KAFKA_BROKER")
	topic := os.Getenv("KAFKA_TOPIC")
	dsn := os.Getenv("TIMESCALE_DSN")
	symbols := config.Symbols(os.Getenv("SYMBOLS"))
	if len(symbols) == 0 {
		log.Fatal("SYMBOLS is required (comma-separated, e.g. BTCUSDT,ETHUSDT)")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal("db:", err)
	}
	defer pool.Close()

	// One goroutine per symbol/partition. Each gets its own kafka.Reader (all
	// sharing GroupID "analytics") and its own StreamProcessor, since
	// StreamProcessor isn't safe for concurrent use. Kafka's group
	// coordinator — not this loop — decides which partition(s), and thus
	// which symbol(s) via the producer's hash-by-key routing, each goroutine
	// actually receives; that assignment can also shift on rebalance, which
	// is why each StreamProcessor must be able to hold more than one
	// symbol's windows (it already does — see stream.go's windows map).
	var wg sync.WaitGroup
	for id := range len(symbols) {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			proc := consumer.NewStreamProcessor(func() []consumer.WindowAggregator {
				return []consumer.WindowAggregator{consumer.NewOHLCVAggregator(pool)}
			})
			c := consumer.New(brokerAddr, topic, proc)
			if err := c.Run(ctx); err != nil {
				log.Printf("consumer[%d]: %v", id, err)
				return
			}
			log.Printf("consumer[%d] stopped", id)
		}(id)
	}
	wg.Wait()
	log.Println("all consumers stopped")
}
