package main

import (
	"context"
	"paymentfc/cmd/payment/handler"
	"paymentfc/cmd/payment/repository"
	"paymentfc/cmd/payment/resource"
	"paymentfc/cmd/payment/service"
	"paymentfc/cmd/payment/usecase"
	"paymentfc/config"
	"paymentfc/infrastructure/constant"
	"paymentfc/infrastructure/log"
	"paymentfc/kafka"
	"paymentfc/models"
	"paymentfc/routes"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	kafkago "github.com/segmentio/kafka-go"
)

func main() {
	// .env 파일이 있으면 로드 (없어도 에러 안 남)
	godotenv.Load()

	cfg := config.LoadConfig()

	log.SetupLogger()

	db := resource.InitDB(cfg.Database)

	// AutoMigrate: payment, payment_anomalies, failed_events, payment_requests 테이블 자동 생성/업데이트
	if err := db.AutoMigrate(&models.Payment{}, &models.PaymentAnomaly{}, &models.FailedEvent{}, &models.PaymentRequest{}); err != nil {
		log.Logger.Fatal().Err(err).Msg("Failed to migrate database")
	}
	log.Logger.Info().Msg("Database migration completed - payment, payment_anomalies, failed_events, payment_requests tables created")

	// Kafka Writer 생성 (payment.success 토픽 발행용)
	kafkaWriter := &kafkago.Writer{
		Addr:     kafkago.TCP(cfg.Kafka.Broker),
		Topic:    constant.KafkaTopicPaymentSuccess,
		Balancer: &kafkago.LeastBytes{},
	}
	defer kafkaWriter.Close()

	// 의존성 주입
	paymentDatabase := repository.NewPaymentDatabase(db)
	paymentPublisher := repository.NewKafkaPublisher(kafkaWriter)
	xenditClient := repository.NewXenditClient(cfg.Xendit.XenditAPIKey)
	xenditService := service.NewXenditService(paymentDatabase, xenditClient)
	xenditUsecase := usecase.NewXenditUsecase(xenditService)

	paymentService := service.NewPaymentService(paymentDatabase, paymentPublisher)
	paymentUsecase := usecase.NewPaymentUsecase(paymentService)
	paymentRequestService := service.NewPaymentRequestService(paymentDatabase, xenditService)
	paymentRequestUsecase := usecase.NewPaymentRequestUsecase(paymentRequestService)
	paymentHandler := handler.NewPaymentHandler(paymentUsecase, xenditUsecase, cfg.Xendit.XenditWebhookToken)

	scheduler := service.SchedulerService{
		Database:       paymentDatabase,
		Xendit:         xenditClient,
		Publisher:      paymentPublisher,
		PaymentService: paymentService,
	}
	scheduler.StartCheckPendingInvoices()
	scheduler.StartProcessPaymentRequests()

	// order.created 컨슈머: toggle에 따라 실시간 인보이스 vs payment_requests 저장만
	kafka.StartOrderConsumer(cfg.Kafka.Broker, constant.KafkaTopicOrderCreated, func(event models.OrderCreatedEvent) {
		ctx := context.Background()
		if cfg.Toggle.DisableCreateInvoiceDirectly {
			// 배치 방식: 저장만, 인보이스는 배치에서 생성
			if err := paymentRequestUsecase.ProcessPaymentRequest(ctx, event); err != nil {
				log.Logger.Error().Err(err).Msgf("Failed to process payment request for order_id: %d", event.OrderID)
			}
		} else {
			// 실시간: 바로 인보이스 생성
			if _, err := xenditUsecase.CreateInvoice(ctx, event); err != nil {
				log.Logger.Error().Err(err).Msgf("Failed to create invoice for order_id: %d", event.OrderID)
			}
		}
	})

	port := cfg.App.Port
	router := gin.Default()

	// 라우트 설정
	routes.SetupRoutes(router, paymentHandler)

	log.Logger.Info().Msgf("Server is running on port %s", port)
	router.Run(":" + port)
}
