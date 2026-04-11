package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PaymentAuditLog struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrderID    int64              `bson:"order_id" json:"order_id"`
	PaymentID  int64              `bson:"payment_id,omitempty" json:"payment_id,omitempty"`
	UserID     int64              `bson:"user_id,omitempty" json:"user_id,omitempty"`
	ExternalID string             `bson:"external_id,omitempty" json:"external_id,omitempty"`
	Event      string             `bson:"event" json:"event"`
	Actor      string             `bson:"actor" json:"actor"`
	Metadata   any                `bson:"metadata,omitempty" json:"metadata,omitempty"`
	CreateTime time.Time          `bson:"create_time" json:"create_time"`
}

type AuditLogFilter struct {
	Event   string
	Actor   string
	OrderID int64
	UserID  int64
	From    time.Time
	To      time.Time
	Limit   int64
	Cursor  string
}

type AuditLogPage struct {
	Logs       []PaymentAuditLog `json:"logs"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

type AuditDailyReportItem struct {
	Date  string `bson:"_id" json:"date"`
	Event string `bson:"event" json:"event"`
	Count int64  `bson:"count" json:"count"`
}
