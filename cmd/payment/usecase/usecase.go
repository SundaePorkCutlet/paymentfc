package usecase

import (
	"context"
	"fmt"
	"paymentfc/cmd/payment/service"
	"paymentfc/infrastructure/constant"
	"paymentfc/infrastructure/log"
	"paymentfc/models"
	"strconv"
	"strings"
)

type PaymentUsecase interface {
	ProcessPaymentSuccess(ctx context.Context, orderID int64) error
	ProcessPaymentWebhook(ctx context.Context, payload models.XenditWebhookPayload) error
}

type paymentUsecase struct {
	paymentService service.PaymentService
}

func NewPaymentUsecase(paymentService service.PaymentService) PaymentUsecase {
	return &paymentUsecase{paymentService: paymentService}
}

func (u *paymentUsecase) ProcessPaymentSuccess(ctx context.Context, orderID int64) error {
	return u.paymentService.ProcessPaymentSuccess(ctx, orderID)
}

// extractOrderID extracts order ID from external ID (e.g. "order-123" -> 123)
func extractOrderID(externalID string) (int64, error) {
	orderIDStr := strings.TrimPrefix(externalID, "order-")
	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse order_id from external_id: %s", externalID)
	}
	return orderID, nil
}

func (u *paymentUsecase) ProcessPaymentWebhook(ctx context.Context, payload models.XenditWebhookPayload) error {
	switch payload.Status {
	case constant.PaymentStatusPaid:
		orderID, err := extractOrderID(payload.ExternalID)
		if err != nil {
			log.Logger.Error().Err(err).Msgf("Failed to extract order ID from external_id: %s", payload.ExternalID)
			return err
		}
		return u.paymentService.ProcessPaymentSuccess(ctx, orderID)
	case constant.PaymentStatusFailed:
		// TODO: 결제 실패 처리
	case constant.PaymentStatusPending:
		// TODO: 결제 대기 처리
	default:
		log.Logger.Error().Msgf("Unknown webhook status: %s", payload.Status)
	}
	return nil
}
