package repository

import (
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type PaymentRepository struct {
	Database *gorm.DB
	Redis    *redis.Client
}

func NewPaymentRepository(db *gorm.DB, redis *redis.Client) *PaymentRepository {
	return &PaymentRepository{
		Database: db,
		Redis:    redis,
	}
}
