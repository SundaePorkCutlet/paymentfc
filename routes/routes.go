package routes

import (
	"net/http"
	"paymentfc/cmd/payment/handler"
	"paymentfc/cmd/payment/resource"
	"paymentfc/config"
	"paymentfc/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRoutes(router *gin.Engine, paymentHandler *handler.PaymentHandler) {
	router.Use(middleware.RequestLogger())
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	router.GET("/ping", paymentHandler.Ping())
	router.POST("/v1/payment/webhook", paymentHandler.HandleXenditWebhook)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "paymentfc",
		})
	})

	router.GET("/debug/queries", func(c *gin.Context) {
		if resource.DBMonitor == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "monitor not initialized"})
			return
		}
		c.JSON(http.StatusOK, resource.DBMonitor.GetDebugInfo())
	})
	router.GET("/debug/mongo/audit-logs", paymentHandler.HandleAuditLogs)
	router.GET("/debug/mongo/audit-report/daily", paymentHandler.HandleAuditDailyReport)
	router.GET("/debug/mongo/stream", paymentHandler.HandleAuditLogStream)

	router.GET("/debug/kafka", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":           "paymentfc",
			"messages_produced": 0,
			"messages_consumed": 0,
			"dlq_count":         0,
			"consumer_stats":    gin.H{},
		})
	})

	private := router.Group("/api")
	private.Use(middleware.AuthMiddleware(config.GetJwtSecret()))
	{
		private.POST("/v1/payment/invoice", paymentHandler.CreateInvoice)
		private.GET("/v1/invoice/:order_id/pdf", paymentHandler.HandleDownloadInvoicePdf)
		private.GET("/v1/failed_payments", paymentHandler.HandleFailedPayments)
		private.GET("/v1/audit-logs", paymentHandler.HandleAuditLogs)
		private.GET("/v1/audit-report/daily", paymentHandler.HandleAuditDailyReport)
	}
}
