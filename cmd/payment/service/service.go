package service

import (
	"context"
	"fmt"
	"math"
	"paymentfc/cmd/payment/repository"
	"paymentfc/constant"
	"paymentfc/log"
	"paymentfc/models"
	"time"
)

type PaymentService interface {
	ProcessPaymentSuccess(ctx context.Context, orderID int64) error
	IsAlreadyPaid(ctx context.Context, orderID int64) (bool, error)
	GetAmountByOrderID(ctx context.Context, orderID int64) (float64, error)
	SavePaymentAnomaly(ctx context.Context, param *models.PaymentAnomaly) error
	SaveFailedPublishEvent(ctx context.Context, param *models.FailedEvent) error
	GetPaymentByOrderID(ctx context.Context, orderID int64) (*models.Payment, error)
	SavePaymentRequestFromEvent(ctx context.Context, event models.OrderCreatedEvent) error
	ProcessBatch(ctx context.Context) error
}

type paymentService struct {
	database      repository.PaymentDatabase
	publisher     repository.PaymentEventPublisher
	xenditService XenditService
	auditLog      repository.AuditLogRepository
}

func NewPaymentService(db repository.PaymentDatabase, publisher repository.PaymentEventPublisher, xenditService XenditService, auditLog repository.AuditLogRepository) PaymentService {
	return &paymentService{
		database:      db,
		publisher:     publisher,
		xenditService: xenditService,
		auditLog:      auditLog,
	}
}

func (s *paymentService) ProcessPaymentSuccess(ctx context.Context, orderID int64) error {
	paid, err := s.database.IsAlreadyPaid(ctx, orderID)
	if err != nil {
		return err
	}
	if paid {
		log.Logger.Info().Int64("order_id", orderID).Msg("Payment already processed, skipping")
		return nil
	}

	// publish event kafka
	err = s.RetryPublishPayment(constant.MaxRetryPublish, func() error {
		if err := s.auditLog.SaveAuditLog(ctx, &models.PaymentAuditLog{
			OrderID: orderID,
			Event:   "PUBLISH_PAYMENT_SUCCESS",
			Actor:   "payment",
		}); err != nil {
			log.Logger.Error().Err(err).Int64("order_id", orderID).Msg("Failed to save audit log")
		}
		return s.publisher.PublishPaymentSuccess(orderID)
	})
	if err != nil {
		log.Logger.Error().Err(err).Int64("order_id", orderID).Msg("s.publisher.PublishPaymentSuccess() got error")
		failed := &models.FailedEvent{
			OrderID:    orderID,
			ExternalID: fmt.Sprintf("order-%d", orderID),
			FailedType: constant.FailedPublishEventPaymentSuccess,
			Notes:      err.Error(),
			Status:     constant.FailedPublishEventStatusNeedToCheck,
			UpdateTime: time.Now(),
		}
		if saveErr := s.database.SaveFailedPublishEvent(ctx, failed); saveErr != nil {
			log.Logger.Error().Err(saveErr).Int64("order_id", orderID).Msg("Failed to save failed_event")
		}
		return err
	}

	err = s.database.MarkPaid(orderID)
	if err != nil {
		return err
	} else {
		err := s.auditLog.SaveAuditLog(ctx, &models.PaymentAuditLog{
			OrderID: orderID,
			Event:   "MARK_PAID",
			Actor:   "payment",
		})
		if err != nil {
			log.Logger.Error().Err(err).Int64("order_id", orderID).Msg("Failed to save audit log")
		}
	}

	return nil
}

func (s *paymentService) IsAlreadyPaid(ctx context.Context, orderID int64) (bool, error) {
	return s.database.IsAlreadyPaid(ctx, orderID)
}

func (s *paymentService) GetAmountByOrderID(ctx context.Context, orderID int64) (float64, error) {
	payment, err := s.database.GetPaymentByOrderID(ctx, orderID)
	if err != nil {
		return 0, err
	}
	return payment.Amount, nil
}

func (s *paymentService) SavePaymentAnomaly(ctx context.Context, param *models.PaymentAnomaly) error {
	if err := s.database.SavePaymentAnomaly(ctx, param); err != nil {
		return err
	}
	s.auditLog.SaveAuditLog(ctx, &models.PaymentAuditLog{
		OrderID:    param.OrderID,
		ExternalID: param.ExternalID,
		Event:      "PAYMENT_ANOMALY_DETECTED",
		Actor:      "webhook_handler",
		Metadata: map[string]any{
			"anomaly_type": param.AnomalyType,
			"notes":        param.Notes,
		},
	})
	return nil
}

func (s *paymentService) SaveFailedPublishEvent(ctx context.Context, param *models.FailedEvent) error {
	if err := s.database.SaveFailedPublishEvent(ctx, param); err != nil {
		return err
	}
	s.auditLog.SaveAuditLog(ctx, &models.PaymentAuditLog{
		OrderID:    param.OrderID,
		ExternalID: param.ExternalID,
		Event:      "FAILED_PUBLISH_EVENT",
		Actor:      "payment_service",
		Metadata: map[string]any{
			"failed_type": param.FailedType,
			"notes":       param.Notes,
		},
	})
	return nil
}

// RetryPublishPayment runs fn up to max times with exponential backoff (2^i seconds); returns nil on first success or the last error.
func (s *paymentService) RetryPublishPayment(max int, fn func() error) error {
	var err error
	for i := 0; i < max; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		wait := time.Duration(math.Pow(2, float64(i))) * time.Second
		log.Logger.Warn().Err(err).Int("retry", i+1).Str("wait", wait.String()).Msg("Kafka publish failed, retrying")
		if i < max-1 {
			time.Sleep(wait)
		}
	}
	return err
}

func (s *paymentService) SavePaymentRequestFromEvent(ctx context.Context, event models.OrderCreatedEvent) error {
	pr := &models.PaymentRequest{
		OrderID:    event.OrderID,
		UserID:     event.UserID,
		Amount:     event.TotalAmount,
		UserEmail:  "",
		Status:     constant.PaymentStatusPending,
		RetryCount: 0,
	}
	if err := s.database.SavePaymentRequest(ctx, pr); err != nil {
		return err
	}

	s.auditLog.SaveAuditLog(ctx, &models.PaymentAuditLog{
		OrderID: event.OrderID,
		UserID:  event.UserID,
		Event:   "PAYMENT_REQUEST_CREATED",
		Actor:   "order_consumer",
		Metadata: map[string]any{
			"amount": event.TotalAmount,
		},
	})

	log.Logger.Info().Int64("order_id", event.OrderID).Msg("Saved payment_request from order.created")
	return nil
}

func (s *paymentService) ProcessBatch(ctx context.Context) error {
	list, err := s.database.GetPendingPaymentRequests(ctx)
	if err != nil {
		return err
	}
	for _, pr := range list {
		invoiceResp, err := s.xenditService.CreateInvoiceFromPaymentRequest(ctx, &pr)
		if err != nil {
			s.auditLog.SaveAuditLog(ctx, &models.PaymentAuditLog{
				OrderID: pr.OrderID,
				UserID:  pr.UserID,
				Event:   "INVOICE_CREATION_FAILED",
				Actor:   "batch_processor",
				Metadata: map[string]any{
					"error": err.Error(),
				},
			})
			if updateErr := s.database.UpdateFailedPaymentRequest(ctx, pr.ID, err.Error()); updateErr != nil {
				log.Logger.Error().Err(updateErr).Int64("payment_request_id", pr.ID).Msg("Failed to update payment_request as failed")
			}
			continue
		}
		s.auditLog.SaveAuditLog(ctx, &models.PaymentAuditLog{
			OrderID:    pr.OrderID,
			UserID:     pr.UserID,
			ExternalID: fmt.Sprintf("order-%d", pr.OrderID),
			Event:      "INVOICE_CREATED",
			Actor:      "batch_processor",
			Metadata: map[string]any{
				"amount":     pr.Amount,
				"invoice_id": invoiceResp.ID,
			},
		})
		if err := s.database.UpdateSuccessPaymentRequest(ctx, pr.ID); err != nil {
			log.Logger.Error().Err(err).Int64("payment_request_id", pr.ID).Msg("Failed to update payment_request as success")
		}
	}
	return nil
}

func (s *paymentService) GetPaymentByOrderID(ctx context.Context, orderID int64) (*models.Payment, error) {
	return s.database.GetPaymentByOrderID(ctx, orderID)
}
