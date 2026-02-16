package service

import (
	"paymentfc/cmd/payment/repository"
)

type PaymentService struct {
	PaymentRepo repository.PaymentRepository
}

func NewPaymentService(paymentRepo repository.PaymentRepository) *PaymentService {
	return &PaymentService{PaymentRepo: paymentRepo}
}
