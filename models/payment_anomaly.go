package models

import "time"

// PaymentAnomaly 결제 이상 건 (금액 불일치 등) 수동 확인용
type PaymentAnomaly struct {
	ID         int64     `json:"id" gorm:"primaryKey;autoIncrement;type:bigserial"`
	OrderID    int64     `json:"order_id" gorm:"type:bigint"`
	ExternalID string    `json:"external_id" gorm:"type:text"`
	AnomalyType int      `json:"anomaly_type" gorm:"type:integer"`
	Notes      string    `json:"notes" gorm:"type:text"`
	Status     int       `json:"status" gorm:"type:integer"`
	CreateTime time.Time `json:"create_time" gorm:"type:timestamp;default:CURRENT_TIMESTAMP"`
	UpdateTime time.Time `json:"update_time" gorm:"type:timestamp"`
}
