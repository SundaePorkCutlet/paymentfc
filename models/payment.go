package models

import "time"

type Payment struct {
	ID         int64     `json:"id" gorm:"primaryKey;autoIncrement;type:bigserial"`
	OrderID    int64     `json:"order_id" gorm:"type:bigint"`
	UserID     int64     `json:"user_id" gorm:"type:bigint"`
	ExternalID string    `json:"external_id" gorm:"type:text;uniqueIndex;not null"`
	Amount     float64   `json:"amount" gorm:"type:numeric"`
	Status     string    `json:"status" gorm:"type:varchar"`
	CreateTime time.Time `json:"create_time" gorm:"type:timestamp;default:CURRENT_TIMESTAMP"`
}

// OrderCreatedEvent ORDERFC에서 Kafka로 발행하는 이벤트 구조체
type OrderCreatedEvent struct {
	OrderID         int64   `json:"order_id"`
	UserID          int64   `json:"user_id"`
	TotalAmount     float64 `json:"total_amount"`
	PaymentMethod   string  `json:"payment_method"`
	ShippingAddress string  `json:"shipping_address"`
}
