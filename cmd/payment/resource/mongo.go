package resource

import (
	"context"
	"paymentfc/config"
	"paymentfc/log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func InitMongo(cfg config.MongoConfig) *mongo.Database {
	if cfg.URI == "" {
		log.Logger.Warn().Msg("MongoDB URI not configured, audit logging disabled")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(cfg.URI)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		log.Logger.Fatal().Err(err).Msg("Failed to connect to MongoDB")
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Logger.Fatal().Err(err).Msg("Failed to ping MongoDB")
	}

	dbName := cfg.Database
	if dbName == "" {
		dbName = "payment_audit"
	}

	log.Logger.Info().Str("database", dbName).Msg("Connected to MongoDB")
	return client.Database(dbName)
}
