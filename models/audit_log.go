package models

import "time"

type PaymentAuditLog struct {
	ID         string    `bson:"_id,omitempty"`
	OrderID    int64     `bson:"order_id"`
	PaymentID  int64     `bson:"payment_id,omitempty"`
	UserID     int64     `bson:"user_id,omitempty"`
	ExternalID string    `bson:"external_id,omitempty"`
	Event      string    `bson:"event"`
	Actor      string    `bson:"actor"`
	Metadata   any       `bson:"metadata,omitempty"`
	CreateTime time.Time `bson:"create_time"`
}
