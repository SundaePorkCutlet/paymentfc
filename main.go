package main

import (
	"paymentfc/cmd/payment/handler"
	"paymentfc/cmd/payment/repository"
	"paymentfc/cmd/payment/resource"
	"paymentfc/cmd/payment/service"
	"paymentfc/cmd/payment/usecase"
	"paymentfc/config"
	"paymentfc/infrastructure/log"
	"paymentfc/models"
	"paymentfc/routes"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/segmentio/kafka-go"
)

func main() {
	// .env 파일이 있으면 로드 (없어도 에러 안 남)
	godotenv.Load()

	cfg := config.LoadConfig()

	log.SetupLogger()

	db := resource.InitDB(cfg.Database)

	// AutoMigrate: payment 테이블 자동 생성/업데이트
	if err := db.AutoMigrate(&models.Payment{}); err != nil {
		log.Logger.Fatal().Err(err).Msg("Failed to migrate database")
	}
	log.Logger.Info().Msg("Database migration completed - payment table created")

	// Kafka Writer 생성 (payment.success 토픽 발행용)
	kafkaWriter := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Kafka.Broker),
		Topic:    cfg.Kafka.Topics[1]["payment.success"],
		Balancer: &kafka.LeastBytes{},
	}
	defer kafkaWriter.Close()

	// 의존성 주입
	paymentDatabase := repository.NewPaymentDatabase(db)
	paymentPublisher := repository.NewKafkaPublisher(kafkaWriter)
	paymentService := service.NewPaymentService(paymentDatabase, paymentPublisher)
	paymentUsecase := usecase.NewPaymentUsecase(paymentService)
	paymentHandler := handler.NewPaymentHandler(paymentUsecase)

	port := cfg.App.Port
	router := gin.Default()

	// 라우트 설정
	routes.SetupRoutes(router, paymentHandler)

	log.Logger.Info().Msgf("Server is running on port %s", port)
	router.Run(":" + port)
}
