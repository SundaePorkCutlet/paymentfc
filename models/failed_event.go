package models

import "time"

// FailedEvent 이벤트 처리 실패 건 (Kafka 발행 실패 등) 재처리/수동 확인용
type FailedEvent struct {
	ID         int64     `json:"id" gorm:"primaryKey;autoIncrement;type:bigserial"`
	OrderID    int64     `json:"order_id" gorm:"type:bigint"`
	ExternalID string    `json:"external_id" gorm:"type:text"`
	FailedType int       `json:"failed_type" gorm:"type:integer"`
	Notes      string    `json:"notes" gorm:"type:text"`
	Status     int       `json:"status" gorm:"type:integer"`
	CreateTime time.Time `json:"create_time" gorm:"type:timestamp;default:CURRENT_TIMESTAMP"`
	UpdateTime time.Time `json:"update_time" gorm:"type:timestamp"`
}
