package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
)

type PaymentEventPublisher interface {
	PublishPaymentStatus(ctx context.Context, orderID int64, status string, topic string) error
}

type kafkaPublisher struct {
	writer *kafka.Writer
}

// NewKafkaPublisher new kafka publisher by given writer pointer of kafka.Writer.
//
// It returns PaymentEventPublisher when successful.
// Otherwise, empty PaymentEventPublisher will be returned.
func NewKafkaPublisher(writer *kafka.Writer) PaymentEventPublisher {
	return &kafkaPublisher{
		writer: writer,
	}
}

// PublishPaymentStatus publishes payment status event to kafka (e.g. "paid", "failed").
func (k *kafkaPublisher) PublishPaymentStatus(ctx context.Context, orderID int64, status string, topic string) error {
	payload := map[string]interface{}{
		"order_id": orderID,
		"status":   status,
		"topic":    topic,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return k.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(fmt.Sprintf("order-%d", orderID)),
		Value: data,
	})
}
