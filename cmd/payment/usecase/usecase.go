package usecase

import (
	"paymentfc/cmd/payment/service"
	"paymentfc/kafka"
)

type PaymentUsecase struct {
	PaymentService service.PaymentService
	KafkaConsumer  *kafka.KafkaConsumer
}

func NewPaymentUsecase(paymentService service.PaymentService, kafkaConsumer *kafka.KafkaConsumer) *PaymentUsecase {
	return &PaymentUsecase{PaymentService: paymentService, KafkaConsumer: kafkaConsumer}
}
