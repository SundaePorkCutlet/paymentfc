package service

import (
	"context"
	"errors"
	"paymentfc/mocks"
	"paymentfc/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestPaymentService_ProcessPaymentSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockPaymentDatabase(ctrl)
	mockPublisher := mocks.NewMockPaymentEventPublisher(ctrl)
	mockAuditLog := mocks.NewMockAuditLogRepository(ctrl)
	mockXenditService := NewMockXenditService(ctrl)

	svc := NewPaymentService(mockDB, mockPublisher, mockXenditService, mockAuditLog)
	ctx := context.Background()
	orderID := int64(12345)

	t.Run("already paid - should skip", func(t *testing.T) {
		mockDB.EXPECT().IsAlreadyPaid(ctx, orderID).Return(true, nil)

		err := svc.ProcessPaymentSuccess(ctx, orderID)
		assert.NoError(t, err)
	})

	t.Run("success flow", func(t *testing.T) {
		mockDB.EXPECT().IsAlreadyPaid(ctx, orderID).Return(false, nil)
		mockAuditLog.EXPECT().SaveAuditLog(ctx, gomock.Any()).Return(nil)
		mockPublisher.EXPECT().PublishPaymentSuccess(orderID).Return(nil)
		mockDB.EXPECT().MarkPaid(orderID).Return(nil)
		mockAuditLog.EXPECT().SaveAuditLog(ctx, gomock.Any()).Return(nil)

		err := svc.ProcessPaymentSuccess(ctx, orderID)
		assert.NoError(t, err)
	})

	t.Run("db error on IsAlreadyPaid", func(t *testing.T) {
		mockDB.EXPECT().IsAlreadyPaid(ctx, orderID).Return(false, errors.New("db error"))

		err := svc.ProcessPaymentSuccess(ctx, orderID)
		assert.Error(t, err)
	})
}

func TestPaymentService_IsAlreadyPaid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockPaymentDatabase(ctrl)
	mockPublisher := mocks.NewMockPaymentEventPublisher(ctrl)
	mockAuditLog := mocks.NewMockAuditLogRepository(ctrl)
	mockXenditService := NewMockXenditService(ctrl)

	svc := NewPaymentService(mockDB, mockPublisher, mockXenditService, mockAuditLog)
	ctx := context.Background()
	orderID := int64(12345)

	t.Run("returns true when paid", func(t *testing.T) {
		mockDB.EXPECT().IsAlreadyPaid(ctx, orderID).Return(true, nil)

		paid, err := svc.IsAlreadyPaid(ctx, orderID)
		assert.NoError(t, err)
		assert.True(t, paid)
	})

	t.Run("returns false when not paid", func(t *testing.T) {
		mockDB.EXPECT().IsAlreadyPaid(ctx, orderID).Return(false, nil)

		paid, err := svc.IsAlreadyPaid(ctx, orderID)
		assert.NoError(t, err)
		assert.False(t, paid)
	})
}

func TestPaymentService_GetAmountByOrderID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockPaymentDatabase(ctrl)
	mockPublisher := mocks.NewMockPaymentEventPublisher(ctrl)
	mockAuditLog := mocks.NewMockAuditLogRepository(ctrl)
	mockXenditService := NewMockXenditService(ctrl)

	svc := NewPaymentService(mockDB, mockPublisher, mockXenditService, mockAuditLog)
	ctx := context.Background()
	orderID := int64(12345)

	t.Run("returns amount successfully", func(t *testing.T) {
		expectedAmount := 100000.0
		mockDB.EXPECT().GetPaymentByOrderID(ctx, orderID).Return(&models.Payment{
			OrderID: orderID,
			Amount:  expectedAmount,
		}, nil)

		amount, err := svc.GetAmountByOrderID(ctx, orderID)
		assert.NoError(t, err)
		assert.Equal(t, expectedAmount, amount)
	})

	t.Run("returns error when payment not found", func(t *testing.T) {
		mockDB.EXPECT().GetPaymentByOrderID(ctx, orderID).Return(nil, errors.New("not found"))

		_, err := svc.GetAmountByOrderID(ctx, orderID)
		assert.Error(t, err)
	})
}

func TestPaymentService_SavePaymentAnomaly(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockPaymentDatabase(ctrl)
	mockPublisher := mocks.NewMockPaymentEventPublisher(ctrl)
	mockAuditLog := mocks.NewMockAuditLogRepository(ctrl)
	mockXenditService := NewMockXenditService(ctrl)

	svc := NewPaymentService(mockDB, mockPublisher, mockXenditService, mockAuditLog)
	ctx := context.Background()

	anomaly := &models.PaymentAnomaly{
		OrderID:     12345,
		ExternalID:  "order-12345",
		AnomalyType: 1,
		Notes:       "Expected 100000, got 90000",
	}

	t.Run("saves anomaly successfully", func(t *testing.T) {
		mockDB.EXPECT().SavePaymentAnomaly(ctx, anomaly).Return(nil)
		mockAuditLog.EXPECT().SaveAuditLog(ctx, gomock.Any()).Return(nil)

		err := svc.SavePaymentAnomaly(ctx, anomaly)
		assert.NoError(t, err)
	})
}

func TestPaymentService_SavePaymentRequestFromEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockPaymentDatabase(ctrl)
	mockPublisher := mocks.NewMockPaymentEventPublisher(ctrl)
	mockAuditLog := mocks.NewMockAuditLogRepository(ctrl)
	mockXenditService := NewMockXenditService(ctrl)

	svc := NewPaymentService(mockDB, mockPublisher, mockXenditService, mockAuditLog)
	ctx := context.Background()

	event := models.OrderCreatedEvent{
		OrderID:     12345,
		UserID:      100,
		TotalAmount: 50000,
	}

	t.Run("saves payment request from event", func(t *testing.T) {
		mockDB.EXPECT().SavePaymentRequest(ctx, gomock.Any()).Return(nil)
		mockAuditLog.EXPECT().SaveAuditLog(ctx, gomock.Any()).Return(nil)

		err := svc.SavePaymentRequestFromEvent(ctx, event)
		assert.NoError(t, err)
	})

	t.Run("returns error when save fails", func(t *testing.T) {
		mockDB.EXPECT().SavePaymentRequest(ctx, gomock.Any()).Return(errors.New("db error"))

		err := svc.SavePaymentRequestFromEvent(ctx, event)
		assert.Error(t, err)
	})
}

