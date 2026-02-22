package repository

import (
	"context"
	"paymentfc/log"
	"paymentfc/models"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

type AuditLogRepository interface {
	SaveAuditLog(ctx context.Context, entry *models.PaymentAuditLog) error
}

type auditLogRepository struct {
	collection *mongo.Collection
}

func NewAuditLogRepository(db *mongo.Database) AuditLogRepository {
	if db == nil {
		return &noopAuditLogRepository{}
	}
	return &auditLogRepository{
		collection: db.Collection("payment_audit_logs"),
	}
}

func (r *auditLogRepository) SaveAuditLog(ctx context.Context, entry *models.PaymentAuditLog) error {
	if entry.CreateTime.IsZero() {
		entry.CreateTime = time.Now()
	}
	_, err := r.collection.InsertOne(ctx, entry)
	if err != nil {
		log.Logger.Error().Err(err).Int64("order_id", entry.OrderID).Str("event", entry.Event).Msg("Failed to insert audit log")
		return err
	}
	return nil
}

type noopAuditLogRepository struct{}

func (r *noopAuditLogRepository) SaveAuditLog(ctx context.Context, entry *models.PaymentAuditLog) error {
	return nil
}
