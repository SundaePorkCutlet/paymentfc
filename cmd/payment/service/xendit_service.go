package service

import (
	"context"
	"fmt"
	"time"
	"paymentfc/cmd/payment/repository"
	"paymentfc/infrastructure/constant"
	"paymentfc/infrastructure/log"
	"paymentfc/models"
)

type XenditService interface {
	CreateInvoice(ctx context.Context, param models.OrderCreatedEvent) (*models.XenditInvoiceResponse, error)
	CheckInvoiceStatus(ctx context.Context, externalID string) (string, error)
}

type xenditService struct {
	database repository.PaymentDatabase
	xendit   repository.XenditClient
}

func NewXenditService(database repository.PaymentDatabase, xenditClient repository.XenditClient) XenditService {
	return &xenditService{
		database: database,
		xendit:   xenditClient,
	}
}

func (s *xenditService) CreateInvoice(ctx context.Context, param models.OrderCreatedEvent) (*models.XenditInvoiceResponse, error) {
	externalID := fmt.Sprintf("order-%d", param.OrderID)
	req := models.XenditInvoiceRequest{
		ExternalID:  externalID,
		Amount:      param.TotalAmount,
		Description: fmt.Sprintf("[FC] Pembayaran Order %d", param.OrderID),
		PayerEmail:  fmt.Sprintf("user%d@test.com", param.UserID),
	}

	resp, err := s.xendit.CreateInvoice(ctx, req)
	if err != nil {
		log.Logger.Error().Err(err).Msgf("Failed to create invoice for order: %d", param.OrderID)
		return nil, err
	}

	payment := &models.Payment{
		OrderID:    param.OrderID,
		UserID:     param.UserID,
		ExternalID: externalID,
		Amount:     param.TotalAmount,
		Status:     constant.PaymentStatusPending,
		CreateTime: time.Now(),
	}
	if err := s.database.SavePayment(ctx, payment); err != nil {
		log.Logger.Error().Err(err).Msgf("Failed to save payment for order: %d", param.OrderID)
		return nil, err
	}

	return resp, nil
}

func (s *xenditService) CheckInvoiceStatus(ctx context.Context, externalID string) (string, error) {
	return s.xendit.CheckInvoiceStatus(ctx, externalID)
}
