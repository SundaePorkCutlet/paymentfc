package service

import (
	"context"
	"paymentfc/cmd/payment/repository"
)

type PaymentService interface {
	ProcessPaymentSuccess(ctx context.Context, orderID int64) error
}

type paymentService struct {
	database  repository.PaymentDatabase
	publisher repository.PaymentEventPublisher
}

func NewPaymentService(db repository.PaymentDatabase, publisher repository.PaymentEventPublisher) PaymentService {
	return &paymentService{
		database:  db,
		publisher: publisher,
	}
}

func (s *paymentService) ProcessPaymentSuccess(ctx context.Context, orderID int64) error {
	err := s.publisher.PublishPaymentSuccess(orderID)
	if err != nil {
		return err
	}

	err = s.database.MarkPaid(orderID)
	if err != nil {
		return err
	}

	return nil
}
