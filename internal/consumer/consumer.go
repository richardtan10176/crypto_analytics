package consumer

import (
	"context"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"

	tradepb "github.com/richardtan10176/crypto_analytics/gen/trade"
)

type Consumer struct {
	reader *kafka.Reader
	proc   *StreamProcessor
}

func New(brokerAddr, topic string, proc *StreamProcessor) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: []string{brokerAddr},
			GroupID: "analytics",
			Topic:   topic,
		}),
		proc: proc,
	}
}

// Run fetches messages until ctx is cancelled, then flushes open windows and
// closes the reader. Offsets are committed explicitly: poison pills (messages
// that fail to unmarshal) are logged and committed so they are never
// reprocessed, while processing errors skip the commit so the message is
// redelivered.
func (c *Consumer) Run(ctx context.Context) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() == nil {
				log.Println("fetch error:", err)
			}
			break
		}

		var event tradepb.TradeEvent
		if err := proto.Unmarshal(msg.Value, &event); err != nil {
			log.Println("poison pill, skipping:", err)
		} else if err := c.proc.Process(ctx, &event); err != nil {
			log.Println("process error:", err)
			continue
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			log.Println("commit error:", err)
		}
	}

	// ctx is already cancelled; give the final flush its own deadline.
	flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.proc.FlushAll(flushCtx); err != nil {
		log.Println("final flush:", err)
	}
	return c.reader.Close()
}
