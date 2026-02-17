package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
)

type PaymentEventPublisher interface {
	PublishPaymentSuccess(orderID int64) error
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

// PublishPaymentSuccess publish payment success event to kafka.
//
// It returns nil error when successful.
// Otherwise, error will be returned.
func (k *kafkaPublisher) PublishPaymentSuccess(orderID int64) error {
	payload := map[string]interface{}{
		"order_id": orderID,
		"status":   "paid",
	}

	data, _ := json.Marshal(payload)
	return k.writer.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(fmt.Sprintf("order-%d", orderID)),
		Value: data,
	})
}
