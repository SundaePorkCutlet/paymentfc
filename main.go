package main

import (
	"paymentfc/cmd/payment/handler"
	"paymentfc/cmd/payment/repository"
	"paymentfc/cmd/payment/resource"
	"paymentfc/cmd/payment/service"
	"paymentfc/cmd/payment/usecase"
	"paymentfc/config"
	"paymentfc/infrastructure/log"
	"paymentfc/kafka"
	"paymentfc/models"
	"paymentfc/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()

	log.SetupLogger()

	redis := resource.InitRedis(cfg.Redis)
	db := resource.InitDB(cfg.Database)

	// AutoMigrate: payment 테이블 자동 생성/업데이트
	if err := db.AutoMigrate(&models.Payment{}); err != nil {
		log.Logger.Fatal().Err(err).Msg("Failed to migrate database")
	}
	log.Logger.Info().Msg("Database migration completed - payment table created")

	kafkaConsumer := kafka.NewKafkaConsumer(cfg.Kafka.Broker, cfg.Kafka.Topics, cfg.Kafka.GroupID)
	defer kafkaConsumer.Close()

	// 의존성 주입
	paymentRepository := repository.NewPaymentRepository(db, redis)
	paymentService := service.NewPaymentService(*paymentRepository)
	paymentUsecase := usecase.NewPaymentUsecase(*paymentService, kafkaConsumer)
	paymentHandler := handler.NewPaymentHandler(*paymentUsecase)

	port := cfg.App.Port
	router := gin.Default()

	// 라우트 설정
	routes.SetupRoutes(router, paymentHandler)

	log.Logger.Info().Msgf("Server is running on port %s", port)
	router.Run(":" + port)
}
