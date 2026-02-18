package routes

import (
	"paymentfc/cmd/payment/handler"
	"paymentfc/config"
	"paymentfc/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine, paymentHandler *handler.PaymentHandler) {
	// 미들웨어 설정
	router.Use(middleware.RequestLogger())

	// public API
	router.GET("/ping", paymentHandler.Ping())
	router.POST("/v1/payment/webhook", paymentHandler.HandleXenditWebhook)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "paymentfc",
		})
	})

	// private API (인증 필요)
	private := router.Group("/api")
	private.Use(middleware.AuthMiddleware(config.GetJwtSecret()))
	{
		private.POST("/v1/payment/invoice", paymentHandler.CreateInvoice)
	}
}
