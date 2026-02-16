package kafka

import (
	"context"
	"paymentfc/infrastructure/log"

	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	reader *kafka.Reader
}

func NewKafkaConsumer(broker string, topics []map[string]string, groupID string) *KafkaConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{broker},
		GroupID: groupID,
	})
	return &KafkaConsumer{reader: reader}
}

func (c *KafkaConsumer) Close() error {
	return c.reader.Close()
}

func (c *KafkaConsumer) ReadMessage(ctx context.Context) (kafka.Message, error) {
	msg, err := c.reader.ReadMessage(ctx)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to read message from Kafka")
		return kafka.Message{}, err
	}
	return msg, nil
}
