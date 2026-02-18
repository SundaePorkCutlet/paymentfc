package repository

import (
	"context"
	"paymentfc/infrastructure/constant"
	"paymentfc/infrastructure/log"
	"paymentfc/models"

	"gorm.io/gorm"
)

type PaymentDatabase interface {
	SavePayment(ctx context.Context, param *models.Payment) error
	SavePaymentAnomaly(ctx context.Context, param *models.PaymentAnomaly) error
	SaveFailedPublishEvent(ctx context.Context, param *models.FailedEvent) error
	MarkPaid(orderID int64) error
	IsAlreadyPaid(ctx context.Context, orderID int64) (bool, error)
	GetPaymentByOrderID(ctx context.Context, orderID int64) (*models.Payment, error)
}

type paymentDatabase struct {
	DB *gorm.DB
}

// NewPaymentDatabase new payment database by given db pointer of gorm.DB.
//
// It returns PaymentDatabase when successful.
// Otherwise, empty PaymentDatabase will be returned.
func NewPaymentDatabase(db *gorm.DB) PaymentDatabase {
	return &paymentDatabase{
		DB: db,
	}
}

func (p *paymentDatabase) SavePayment(ctx context.Context, param *models.Payment) error {
	if err := p.DB.WithContext(ctx).Table("payments").Create(param).Error; err != nil {
		log.Logger.Error().Err(err).Msgf("Failed to save payment for order_id: %d", param.OrderID)
		return err
	}
	return nil
}

func (p *paymentDatabase) SavePaymentAnomaly(ctx context.Context, param *models.PaymentAnomaly) error {
	if err := p.DB.WithContext(ctx).Table("payment_anomalies").Create(param).Error; err != nil {
		log.Logger.Error().Err(err).Msgf("Failed to save payment anomaly for order_id: %d", param.OrderID)
		return err
	}
	return nil
}

func (p *paymentDatabase) SaveFailedPublishEvent(ctx context.Context, param *models.FailedEvent) error {
	if err := p.DB.WithContext(ctx).Table("failed_events").Create(param).Error; err != nil {
		log.Logger.Error().Err(err).Msgf("Failed to save failed_event for order_id: %d", param.OrderID)
		return err
	}
	return nil
}

func (p *paymentDatabase) MarkPaid(orderID int64) error {
	err := p.DB.Table("payments").Where("order_id = ?", orderID).Update("status", constant.PaymentStatusPaid).Error
	if err != nil {
		log.Logger.Error().Err(err).Msgf("Failed to mark payment as paid for order_id: %d", orderID)
		return err
	}
	return nil
}

func (p *paymentDatabase) IsAlreadyPaid(ctx context.Context, orderID int64) (bool, error) {
	var result models.Payment
	err := p.DB.Table("payments").WithContext(ctx).Where("order_id = ?", orderID).First(&result).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}
	return result.Status == constant.PaymentStatusPaid, nil
}

func (p *paymentDatabase) GetPaymentByOrderID(ctx context.Context, orderID int64) (*models.Payment, error) {
	var result models.Payment
	err := p.DB.Table("payments").WithContext(ctx).Where("order_id = ?", orderID).First(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}
