package models

import "time"

type Payment struct {
	ID          int64     `json:"id" gorm:"primaryKey;autoIncrement;type:bigserial"`
	OrderID     int64     `json:"order_id" gorm:"type:bigint;index:idx_payments_order"`
	UserID      int64     `json:"user_id" gorm:"type:bigint;index:idx_payments_user"`
	ExternalID  string    `json:"external_id" gorm:"type:text;uniqueIndex;not null"`
	Amount      float64   `json:"amount" gorm:"type:numeric"`
	Status      string    `json:"status" gorm:"type:varchar;index:idx_payments_status_time"`
	CreateTime  time.Time `json:"create_time" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;index:idx_payments_status_time"`
	UpdateTime  time.Time `json:"update_time" gorm:"type:timestamp"`
	ExpiredTime time.Time `json:"expired_time" gorm:"type:timestamp"`
}

type PaymentRequest struct {
	ID         int64     `json:"id" gorm:"primaryKey;autoIncrement;type:bigserial"`
	OrderID    int64     `json:"order_id" gorm:"type:bigint;not null;uniqueIndex:idx_payreq_order"`
	UserID     int64     `json:"user_id" gorm:"type:bigint"`
	Amount     float64   `json:"amount" gorm:"type:numeric"`
	UserEmail  string    `json:"user_email" gorm:"type:varchar"`
	Status     string    `json:"status" gorm:"type:varchar;index:idx_payreq_status_time"`
	RetryCount int       `json:"retry_count" gorm:"type:int"`
	Notes      string    `json:"notes" gorm:"type:text"`
	CreateTime time.Time `json:"create_time" gorm:"type:timestamp;autoCreateTime;index:idx_payreq_status_time"`
	UpdateTime time.Time `json:"update_time" gorm:"type:timestamp;autoUpdateTime"`
}

type FailedPaymentList struct {
	TotalFailedPayments int64            `json:"total_failed_payments"`
	PaymentList         []PaymentRequest `json:"payment_list"`
}
