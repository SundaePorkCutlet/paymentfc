package kafka

import (
	"context"
	"encoding/json"
	"paymentfc/log"
	"paymentfc/models"

	"github.com/segmentio/kafka-go"
)

// StartOrderConsumer start order consumer by given broker, topic, and handler.
func StartOrderConsumer(broker, topic string, handler func(models.OrderCreatedEvent)) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{broker},
		Topic:   topic,
		GroupID: "paymentfc",
	})

	go func(r *kafka.Reader) {
		defer r.Close()
		for {
			msg, err := r.ReadMessage(context.Background())
			if err != nil {
				log.Logger.Error().Err(err).Msg("Failed to read order.created message")
				continue
			}

			var event models.OrderCreatedEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Logger.Error().Err(err).Msg("Failed to unmarshal order.created message")
				continue
			}

			handler(event)
		}
	}(reader)
}
