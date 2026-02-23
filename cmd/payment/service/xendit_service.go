package service

import (
	"context"
	"fmt"
	"paymentfc/cmd/payment/repository"
	"paymentfc/constant"
	usergrpc "paymentfc/grpc"
	"paymentfc/log"
	"paymentfc/models"
	"time"
)

type XenditService interface {
	CreateInvoice(ctx context.Context, param models.OrderCreatedEvent) (*models.XenditInvoiceResponse, error)
	CreateInvoiceFromPaymentRequest(ctx context.Context, pr *models.PaymentRequest) (*models.XenditInvoiceResponse, error)
	CheckInvoiceStatus(ctx context.Context, externalID string) (string, error)
}

type xenditService struct {
	database   repository.PaymentDatabase
	xendit     repository.XenditClient
	userClient *usergrpc.UserClient
}

func NewXenditService(database repository.PaymentDatabase, xenditClient repository.XenditClient, userClient *usergrpc.UserClient) XenditService {
	return &xenditService{
		database:   database,
		xendit:     xenditClient,
		userClient: userClient,
	}
}

func (s *xenditService) CreateInvoice(ctx context.Context, param models.OrderCreatedEvent) (*models.XenditInvoiceResponse, error) {
	externalID := fmt.Sprintf("order-%d", param.OrderID)

	if s.userClient == nil {
		return nil, fmt.Errorf("user gRPC client is not initialized")
	}
	userInfo, err := s.userClient.GetUserInfoByUserId(ctx, param.UserID)
	if err != nil {
		log.Logger.Error().Err(err).Int64("user_id", param.UserID).Msg("Failed to get user info via gRPC")
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	payerEmail := userInfo.Email
	req := models.XenditInvoiceRequest{
		ExternalID:  externalID,
		Amount:      param.TotalAmount,
		Description: fmt.Sprintf("[FC] Pembayaran Order %d", param.OrderID),
		PayerEmail:  payerEmail,
	}

	xenditInvoiceInfo, err := s.xendit.CreateInvoice(ctx, req)
	if err != nil {
		log.Logger.Error().Err(err).Msgf("Failed to create invoice for order: %d", param.OrderID)
		return nil, err
	}

	payment := &models.Payment{
		OrderID:     param.OrderID,
		UserID:      param.UserID,
		ExternalID:  externalID,
		Amount:      param.TotalAmount,
		Status:      constant.PaymentStatusPending,
		CreateTime:  time.Now(),
		ExpiredTime: xenditInvoiceInfo.ExpireDate,
	}
	if err := s.database.SavePayment(ctx, payment); err != nil {
		log.Logger.Error().Err(err).Msgf("Failed to save payment for order: %d", param.OrderID)
		return nil, err
	}

	return xenditInvoiceInfo, nil
}

func (s *xenditService) CreateInvoiceFromPaymentRequest(ctx context.Context, pr *models.PaymentRequest) (*models.XenditInvoiceResponse, error) {
	externalID := fmt.Sprintf("order-%d", pr.OrderID)
	payerEmail := pr.UserEmail
	if payerEmail == "" {
		if s.userClient == nil {
			return nil, fmt.Errorf("user gRPC client is not initialized")
		}
		userInfo, err := s.userClient.GetUserInfoByUserId(ctx, pr.UserID)
		if err != nil {
			log.Logger.Error().Err(err).Int64("user_id", pr.UserID).Msg("Failed to get user info via gRPC")
			return nil, fmt.Errorf("failed to get user info: %w", err)
		}
		payerEmail = userInfo.Email
	}
	req := models.XenditInvoiceRequest{
		ExternalID:  externalID,
		Amount:      pr.Amount,
		Description: fmt.Sprintf("[FC] Pembayaran Order %d", pr.OrderID),
		PayerEmail:  payerEmail,
	}

	resp, err := s.xendit.CreateInvoice(ctx, req)
	if err != nil {
		log.Logger.Error().Err(err).Msgf("Failed to create invoice for payment_request order_id: %d", pr.OrderID)
		return nil, err
	}

	payment := &models.Payment{
		OrderID:    pr.OrderID,
		UserID:     pr.UserID,
		ExternalID: externalID,
		Amount:     pr.Amount,
		Status:     constant.PaymentStatusPending,
		CreateTime: time.Now(),
	}
	if err := s.database.SavePayment(ctx, payment); err != nil {
		log.Logger.Error().Err(err).Msgf("Failed to save payment for order_id: %d", pr.OrderID)
		return nil, err
	}

	return resp, nil
}

func (s *xenditService) CheckInvoiceStatus(ctx context.Context, externalID string) (string, error) {
	return s.xendit.CheckInvoiceStatus(ctx, externalID)
}
