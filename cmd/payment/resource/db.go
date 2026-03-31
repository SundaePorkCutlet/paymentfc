package resource

import (
	"fmt"
	"paymentfc/infrastructure/dbmonitor"
	"paymentfc/log"
	"time"

	"paymentfc/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DBMonitor *dbmonitor.Monitor

func InitDB(cfg config.DatabaseConfig) *gorm.DB {

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Logger.Fatal().Err(err).Msg("Failed to connect to database")
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Logger.Fatal().Err(err).Msg("Failed to get underlying sql.DB")
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	DBMonitor = dbmonitor.NewMonitor(100 * time.Millisecond)
	if err := db.Use(DBMonitor); err != nil {
		log.Logger.Warn().Err(err).Msg("Failed to register DB monitor plugin")
	}

	log.Logger.Info().Msg("Connected to database with connection pool configured")
	return db
}
