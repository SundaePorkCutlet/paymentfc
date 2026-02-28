package routes

import (
	"paymentfc/cmd/payment/handler"
	"paymentfc/config"
	"paymentfc/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRoutes(router *gin.Engine, paymentHandler *handler.PaymentHandler) {
	router.Use(middleware.RequestLogger())
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
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
		private.GET("/v1/invoice/:order_id/pdf", paymentHandler.HandleDownloadInvoicePdf)
		private.GET("/v1/failed_payments", paymentHandler.HandleFailedPayments)
	}
}
