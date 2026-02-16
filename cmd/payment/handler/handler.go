package handler

import (
	"net/http"
	"paymentfc/cmd/payment/usecase"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	PaymentUsecase usecase.PaymentUsecase
}

func NewPaymentHandler(paymentUsecase usecase.PaymentUsecase) *PaymentHandler {
	return &PaymentHandler{PaymentUsecase: paymentUsecase}
}

func (h *PaymentHandler) Ping() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	}
}
