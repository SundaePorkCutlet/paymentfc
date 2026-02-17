package kafka

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"
)

type KafkaWriter struct {
	writer *kafka.Writer
}

func NewKafkaWriter(broker string, topic string) *KafkaWriter {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(broker),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	return &KafkaWriter{writer: writer}
}

func (w *KafkaWriter) Close() error {
	return w.writer.Close()
}

func (w *KafkaWriter) WriteMessage(ctx context.Context, key string, value interface{}) error {
	json, err := json.Marshal(value)
	if err != nil {
		return err
	}
	msg := kafka.Message{
		Key:   []byte(key),
		Value: json,
	}
	return w.writer.WriteMessages(ctx, msg)
}
