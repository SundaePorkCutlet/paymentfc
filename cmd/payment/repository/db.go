package repository

import (
	"paymentfc/infrastructure/log"

	"gorm.io/gorm"
)

type PaymentDatabase interface {
	MarkPaid(orderID int64) error
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

func (p *paymentDatabase) MarkPaid(orderID int64) error {
	err := p.DB.Table("payments").Where("order_id = ?", orderID).Update("status", "paid").Error
	if err != nil {
		log.Logger.Error().Err(err).Msgf("Failed to mark payment as paid for order_id: %d", orderID)
		return err
	}
	return nil
}
