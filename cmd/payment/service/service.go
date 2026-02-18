package service

import (
	"context"
	"fmt"
	"math"
	"paymentfc/cmd/payment/repository"
	"paymentfc/infrastructure/constant"
	"paymentfc/infrastructure/log"
	"paymentfc/models"
	"time"
)

type PaymentService interface {
	ProcessPaymentSuccess(ctx context.Context, orderID int64) error
	IsAlreadyPaid(ctx context.Context, orderID int64) (bool, error)
	GetAmountByOrderID(ctx context.Context, orderID int64) (float64, error)
	SavePaymentAnomaly(ctx context.Context, param *models.PaymentAnomaly) error
	SaveFailedPublishEvent(ctx context.Context, param *models.FailedEvent) error
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
	return s.database.SavePaymentAnomaly(ctx, param)
}

func (s *paymentService) SaveFailedPublishEvent(ctx context.Context, param *models.FailedEvent) error {
	return s.database.SaveFailedPublishEvent(ctx, param)
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
