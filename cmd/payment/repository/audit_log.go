package repository

import (
	"context"
	"errors"
	"paymentfc/log"
	"paymentfc/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AuditLogRepository interface {
	SaveAuditLog(ctx context.Context, entry *models.PaymentAuditLog) error
	ListAuditLogs(ctx context.Context, filter models.AuditLogFilter) (models.AuditLogPage, error)
	GetDailyReport(ctx context.Context, from, to time.Time) ([]models.AuditDailyReportItem, error)
	WatchInsertStream(ctx context.Context, out chan<- models.PaymentAuditLog) error
}

type auditLogRepository struct {
	collection *mongo.Collection
}

func NewAuditLogRepository(db *mongo.Database) AuditLogRepository {
	if db == nil {
		return &noopAuditLogRepository{}
	}
	repo := &auditLogRepository{
		collection: db.Collection("payment_audit_logs"),
	}
	repo.ensureIndexes(context.Background())
	return repo
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

func (r *auditLogRepository) ListAuditLogs(ctx context.Context, filter models.AuditLogFilter) (models.AuditLogPage, error) {
	page := models.AuditLogPage{Logs: make([]models.PaymentAuditLog, 0)}
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	q := bson.M{}
	if filter.Event != "" {
		q["event"] = filter.Event
	}
	if filter.Actor != "" {
		q["actor"] = filter.Actor
	}
	if filter.OrderID > 0 {
		q["order_id"] = filter.OrderID
	}
	if filter.UserID > 0 {
		q["user_id"] = filter.UserID
	}
	if !filter.From.IsZero() || !filter.To.IsZero() {
		t := bson.M{}
		if !filter.From.IsZero() {
			t["$gte"] = filter.From
		}
		if !filter.To.IsZero() {
			t["$lte"] = filter.To
		}
		q["create_time"] = t
	}
	if filter.Cursor != "" {
		oid, err := primitive.ObjectIDFromHex(filter.Cursor)
		if err == nil {
			q["_id"] = bson.M{"$lt": oid}
		}
	}

	cur, err := r.collection.Find(ctx, q, options.Find().
		SetSort(bson.D{{Key: "_id", Value: -1}}).
		SetLimit(limit + 1))
	if err != nil {
		return page, err
	}
	defer cur.Close(ctx)

	all := make([]models.PaymentAuditLog, 0)
	if err := cur.All(ctx, &all); err != nil {
		return page, err
	}
	if int64(len(all)) > limit {
		last := all[limit-1]
		page.NextCursor = last.ID.Hex()
		all = all[:limit]
	}
	page.Logs = all
	return page, nil
}

func (r *auditLogRepository) GetDailyReport(ctx context.Context, from, to time.Time) ([]models.AuditDailyReportItem, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"create_time": bson.M{
				"$gte": from,
				"$lte": to,
			},
		}}},
		{{Key: "$project", Value: bson.M{
			"event": 1,
			"date": bson.M{
				"$dateToString": bson.M{
					"format": "%Y-%m-%d",
					"date":   "$create_time",
				},
			},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"date":  "$date",
				"event": "$event",
			},
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$project", Value: bson.M{
			"_id":   "$_id.date",
			"event": "$_id.event",
			"count": 1,
		}}},
		{{Key: "$sort", Value: bson.M{"_id": -1, "event": 1}}},
	}

	cur, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	items := make([]models.AuditDailyReportItem, 0)
	if err := cur.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *auditLogRepository) WatchInsertStream(ctx context.Context, out chan<- models.PaymentAuditLog) error {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"operationType": "insert"}}},
	}
	stream, err := r.collection.Watch(ctx, pipeline)
	if err != nil {
		return err
	}
	defer stream.Close(ctx)

	type changeEvent struct {
		FullDocument models.PaymentAuditLog `bson:"fullDocument"`
	}
	for stream.Next(ctx) {
		var ev changeEvent
		if err := stream.Decode(&ev); err != nil {
			continue
		}
		select {
		case <-ctx.Done():
			return nil
		case out <- ev.FullDocument:
		}
	}
	return stream.Err()
}

func (r *auditLogRepository) ensureIndexes(ctx context.Context) {
	idx := []mongo.IndexModel{
		{Keys: bson.D{{Key: "event", Value: 1}, {Key: "create_time", Value: -1}}},
		{Keys: bson.D{{Key: "order_id", Value: 1}, {Key: "create_time", Value: -1}}},
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "create_time", Value: -1}}},
		{
			Keys: bson.D{{Key: "create_time", Value: 1}},
			Options: options.Index().
				SetExpireAfterSeconds(90 * 24 * 60 * 60).
				SetName("ttl_create_time_90d"),
		},
	}
	if _, err := r.collection.Indexes().CreateMany(ctx, idx); err != nil {
		log.Logger.Warn().Err(err).Msg("Failed to create audit log indexes")
	}
}

type noopAuditLogRepository struct{}

func (r *noopAuditLogRepository) SaveAuditLog(ctx context.Context, entry *models.PaymentAuditLog) error {
	return nil
}

func (r *noopAuditLogRepository) ListAuditLogs(ctx context.Context, filter models.AuditLogFilter) (models.AuditLogPage, error) {
	return models.AuditLogPage{Logs: []models.PaymentAuditLog{}}, nil
}

func (r *noopAuditLogRepository) GetDailyReport(ctx context.Context, from, to time.Time) ([]models.AuditDailyReportItem, error) {
	return []models.AuditDailyReportItem{}, nil
}

func (r *noopAuditLogRepository) WatchInsertStream(ctx context.Context, out chan<- models.PaymentAuditLog) error {
	return errors.New("audit log repository disabled")
}
