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
