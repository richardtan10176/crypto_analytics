package producer

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func New(ctx context.Context, brokerAddr, topic string) (*Producer, error) {
	conn, err := kafka.DialContext(ctx, "tcp", brokerAddr)
	if err != nil {
		return nil, fmt.Errorf("damn we couldnt get the broker: %w", err)
	}
	defer conn.Close()

	err = conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create topic: %w", err)
	}

	return &Producer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokerAddr),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		},
	}, nil
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
