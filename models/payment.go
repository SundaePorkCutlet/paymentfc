package models

import "time"

type Payment struct {
	ID            int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	OrderID       int64     `json:"order_id" gorm:"not null"`
	UserID        int64     `json:"user_id" gorm:"not null"`
	Amount        float64   `json:"amount" gorm:"not null"`
	PaymentMethod string    `json:"payment_method" gorm:"not null"`
	Status        int       `json:"status" gorm:"not null;default:0"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// OrderCreatedEvent ORDERFC에서 Kafka로 발행하는 이벤트 구조체
type OrderCreatedEvent struct {
	OrderID         int64   `json:"order_id"`
	UserID          int64   `json:"user_id"`
	TotalAmount     float64 `json:"total_amount"`
	PaymentMethod   string  `json:"payment_method"`
	ShippingAddress string  `json:"shipping_address"`
}
