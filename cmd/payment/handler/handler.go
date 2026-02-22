package handler

import (
	"net/http"
	"paymentfc/cmd/payment/usecase"
	"paymentfc/log"
	"paymentfc/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	PaymentUsecase     usecase.PaymentUsecase
	XenditUsecase      usecase.XenditUsecase
	XenditWebhookToken string
}

func NewPaymentHandler(paymentUsecase usecase.PaymentUsecase, xenditUsecase usecase.XenditUsecase, xenditWebhookToken string) *PaymentHandler {
	return &PaymentHandler{
		PaymentUsecase:     paymentUsecase,
		XenditUsecase:      xenditUsecase,
		XenditWebhookToken: xenditWebhookToken,
	}
}

func (h *PaymentHandler) Ping() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	}
}

func (h *PaymentHandler) HandleXenditWebhook(c *gin.Context) {
	if c.GetHeader("x-callback-token") != h.XenditWebhookToken {
		log.Logger.Warn().Msg("Xendit webhook: invalid or missing callback token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid callback token"})
		return
	}

	var payload models.XenditWebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Logger.Error().Err(err).Msg("Failed to bind webhook payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.PaymentUsecase.ProcessPaymentWebhook(c.Request.Context(), payload)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to process payment webhook")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "webhook processed"})
}

// CreateInvoice creates a Xendit invoice for the given order and returns invoice_url etc.
func (h *PaymentHandler) CreateInvoice(c *gin.Context) {
	var req models.OrderCreatedEvent
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Logger.Error().Err(err).Msg("Failed to bind create invoice request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.XenditUsecase.CreateInvoice(c.Request.Context(), req)
	if err != nil {
		log.Logger.Error().Err(err).Msgf("Failed to create invoice for order_id: %d", req.OrderID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          resp.ID,
		"invoice_url": resp.InvoiceURL,
		"status":      resp.Status,
	})
}

func (h *PaymentHandler) HandleDownloadInvoicePdf(c *gin.Context) {
	orderIdStr := c.Param("order_id")

	orderIdInt, err := strconv.ParseInt(orderIdStr, 10, 64)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to parse order id")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filePath, err := h.PaymentUsecase.DownloadInvoicePdf(c.Request.Context(), orderIdInt)
	if err != nil {
		log.Logger.Error().Err(err).Msgf("Failed to get payment by order id: %d", orderIdInt)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.FileAttachment(filePath, filePath)
}
