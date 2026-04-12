package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"paymentfc/cmd/payment/usecase"
	bizmetrics "paymentfc/infrastructure/metrics"
	"paymentfc/log"
	"paymentfc/models"
	"strconv"
	"time"

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
		bizmetrics.XenditWebhookProcessed.WithLabelValues("invalid_token").Inc()
		log.Logger.Warn().Msg("Xendit webhook: invalid or missing callback token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid callback token"})
		return
	}

	var payload models.XenditWebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		bizmetrics.XenditWebhookProcessed.WithLabelValues("bind_error").Inc()
		log.Logger.Error().Err(err).Msg("Failed to bind webhook payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.PaymentUsecase.ProcessPaymentWebhook(c.Request.Context(), payload)
	if err != nil {
		bizmetrics.XenditWebhookProcessed.WithLabelValues("process_error").Inc()
		log.Logger.Error().Err(err).Msg("Failed to process payment webhook")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	bizmetrics.XenditWebhookProcessed.WithLabelValues("success").Inc()
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

	c.FileAttachment(filePath, "invoice_"+orderIdStr+".pdf")
}

func (h *PaymentHandler) HandleFailedPayments(c *gin.Context) {
	paymentList, err := h.PaymentUsecase.GetFailedPaymentList(c.Request.Context())
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to get failed payment list")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, paymentList)
}

func (h *PaymentHandler) HandleAuditLogs(c *gin.Context) {
	filter := models.AuditLogFilter{
		Event:  c.Query("event"),
		Actor:  c.Query("actor"),
		Cursor: c.Query("cursor"),
		Limit:  20,
	}
	if orderIDStr := c.Query("order_id"); orderIDStr != "" {
		if v, err := strconv.ParseInt(orderIDStr, 10, 64); err == nil {
			filter.OrderID = v
		}
	}
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if v, err := strconv.ParseInt(userIDStr, 10, 64); err == nil {
			filter.UserID = v
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if v, err := strconv.ParseInt(limitStr, 10, 64); err == nil && v > 0 {
			filter.Limit = v
		}
	}
	if fromStr := c.Query("from"); fromStr != "" {
		if t, err := parseTimeParam(fromStr); err == nil {
			filter.From = t
		}
	}
	if toStr := c.Query("to"); toStr != "" {
		if t, err := parseTimeParam(toStr); err == nil {
			filter.To = t
		}
	}

	page, err := h.PaymentUsecase.ListAuditLogs(c.Request.Context(), filter)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to list audit logs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, page)
}

func (h *PaymentHandler) HandleAuditDailyReport(c *gin.Context) {
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -7)
	to := now

	if fromStr := c.Query("from"); fromStr != "" {
		if t, err := parseTimeParam(fromStr); err == nil {
			from = t
		}
	}
	if toStr := c.Query("to"); toStr != "" {
		if t, err := parseTimeParam(toStr); err == nil {
			to = t
		}
	}
	items, err := h.PaymentUsecase.GetAuditDailyReport(c.Request.Context(), from, to)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to get audit daily report")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"from":  from,
		"to":    to,
		"items": items,
	})
}

func (h *PaymentHandler) HandleAuditLogStream(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming unsupported"})
		return
	}

	events := make(chan models.PaymentAuditLog, 32)
	errCh := make(chan error, 1)

	go func() {
		errCh <- h.PaymentUsecase.WatchAuditInsertStream(c.Request.Context(), events)
	}()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case err := <-errCh:
			// Change Stream은 Replica Set 필요. 단일 노드면 에러를 이벤트로 전달하고 종료.
			payload := map[string]any{
				"time":    time.Now().UTC().Format(time.RFC3339),
				"topic":   "mongo.change_stream.error",
				"payload": gin.H{"error": err.Error()},
			}
			b, _ := json.Marshal(payload)
			fmt.Fprintf(c.Writer, "data: %s\n\n", b)
			flusher.Flush()
			return
		case ev := <-events:
			payload := map[string]any{
				"time":    time.Now().UTC().Format(time.RFC3339),
				"topic":   "payment.audit_log.insert",
				"payload": ev,
			}
			b, _ := json.Marshal(payload)
			fmt.Fprintf(c.Writer, "data: %s\n\n", b)
			flusher.Flush()
		}
	}
}

func parseTimeParam(raw string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("invalid time format: %s", raw)
}
