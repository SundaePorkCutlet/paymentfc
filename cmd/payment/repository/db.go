package repository

import (
	"context"
	"paymentfc/infrastructure/constant"
	"paymentfc/infrastructure/log"
	"paymentfc/models"
	"time"

	"gorm.io/gorm"
)

type PaymentDatabase interface {
	SavePayment(ctx context.Context, param *models.Payment) error
	SavePaymentAnomaly(ctx context.Context, param *models.PaymentAnomaly) error
	SaveFailedPublishEvent(ctx context.Context, param *models.FailedEvent) error
	MarkPaid(orderID int64) error
	IsAlreadyPaid(ctx context.Context, orderID int64) (bool, error)
	GetPaymentByOrderID(ctx context.Context, orderID int64) (*models.Payment, error)
	GetPendingInvoices(ctx context.Context) ([]models.Payment, error)
	SavePaymentRequest(ctx context.Context, param *models.PaymentRequest) error
	GetPendingPaymentRequests(ctx context.Context) ([]models.PaymentRequest, error)
	GetFailedPaymentRequests(ctx context.Context) ([]models.PaymentRequest, error)
	UpdateSuccessPaymentRequest(ctx context.Context, paymentRequestID int64) error
	UpdateFailedPaymentRequest(ctx context.Context, paymentRequestID int64, notes string) error
	UpdatePendingPaymentRequest(ctx context.Context, paymentRequestID int64) error
	GetExpiredPendingPayments(ctx context.Context) ([]models.Payment, error)
	MarkExpired(ctx context.Context, paymentID int64) error
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

func (p *paymentDatabase) GetPendingInvoices(ctx context.Context) ([]models.Payment, error) {
	var result []models.Payment
	err := p.DB.Table("payments").WithContext(ctx).Where("status = ? AND create_time >= now() - interval '1 day'", constant.PaymentStatusPending).Find(&result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
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

func (p *paymentDatabase) SavePaymentRequest(ctx context.Context, param *models.PaymentRequest) error {
	if err := p.DB.WithContext(ctx).Table("payment_requests").Create(param).Error; err != nil {
		log.Logger.Error().Err(err).Msgf("Failed to save payment request for order_id: %d", param.OrderID)
		return err
	}
	return nil
}
func (p *paymentDatabase) GetPendingPaymentRequests(ctx context.Context) ([]models.PaymentRequest, error) {
	var result []models.PaymentRequest
	err := p.DB.Table("payment_requests").WithContext(ctx).Where("status = ?", constant.PaymentStatusPending).Limit(5).Order("create_time ASC").Find(&result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (p *paymentDatabase) GetFailedPaymentRequests(ctx context.Context) ([]models.PaymentRequest, error) {
	var result []models.PaymentRequest
	err := p.DB.Table("payment_requests").WithContext(ctx).Where("status = ? AND retry_count <= ?", constant.PaymentStatusFailed, 3).Limit(5).Order("create_time ASC").Find(&result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (p *paymentDatabase) UpdateSuccessPaymentRequest(ctx context.Context, paymentRequestID int64) error {
	err := p.DB.Table("payment_requests").Where("id = ?", paymentRequestID).Updates(
		map[string]interface{}{
			"status":      constant.PaymentStatusPaid,
			"update_time": time.Now(),
		}).Error
	if err != nil {
		log.Logger.Error().Err(err).Msgf("Failed to update payment request as success for payment_request_id: %d", paymentRequestID)
		return err
	}
	return nil
}

func (p *paymentDatabase) UpdateFailedPaymentRequest(ctx context.Context, paymentRequestID int64, notes string) error {
	err := p.DB.Table("payment_requests").Where("id = ?", paymentRequestID).Updates(
		map[string]interface{}{
			"status":      constant.PaymentStatusFailed,
			"update_time": time.Now(),
			"retry_count": gorm.Expr("retry_count + 1"),
			"notes":       notes,
		}).Error
	if err != nil {
		log.Logger.Error().Err(err).Msgf("Failed to update payment request as failed for payment_request_id: %d", paymentRequestID)
		return err
	}
	return nil
}

func (p *paymentDatabase) UpdatePendingPaymentRequest(ctx context.Context, paymentRequestID int64) error {
	err := p.DB.Table("payment_requests").Where("id = ?", paymentRequestID).Updates(
		map[string]interface{}{
			"status":      constant.PaymentStatusPending,
			"update_time": time.Now(),
		}).Error
	if err != nil {
		log.Logger.Error().Err(err).Msgf("Failed to update payment request as pending for payment_request_id: %d", paymentRequestID)
		return err
	}
	return nil
}

func (p *paymentDatabase) GetExpiredPendingPayments(ctx context.Context) ([]models.Payment, error) {
	var result []models.Payment
	err := p.DB.Table("payments").WithContext(ctx).Where("status = ? AND expired_time < now()", constant.PaymentStatusPending).Find(&result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (p *paymentDatabase) MarkExpired(ctx context.Context, paymentID int64) error {
	err := p.DB.Table("payments").Where("id = ?", paymentID).Updates(
		map[string]interface{}{
			"status":      constant.PaymentStatusExpired,
			"update_time": time.Now(),
		}).Error
	if err != nil {
		log.Logger.Error().Err(err).Int64("payment_id", paymentID).Msg("Failed to mark payment as expired")
		return err
	}
	return nil
}
