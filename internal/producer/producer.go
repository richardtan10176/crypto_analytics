package producer

import (
	"context"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func New(brokerAddr, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokerAddr),
			Topic:                  topic,
			Balancer:               &kafka.LeastBytes{},
			AllowAutoTopicCreation: true,
		},
	}
}

// Publish sends a serialized message to Kafka. key should be the trade symbol
// (e.g. "BTCUSDT") to route all messages for the same symbol to one partition.
func (p *Producer) Publish(ctx context.Context, key string, value []byte) error {
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: value,
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
