package service

import (
	"context"
	"errors"
	"paymentfc/mocks"
	"paymentfc/models"
	pb "paymentfc/pb/proto"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestXenditService_CreateInvoice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockPaymentDatabase(ctrl)
	mockXenditClient := mocks.NewMockXenditClient(ctrl)
	mockUserClient := mocks.NewMockUserClientInterface(ctrl)

	svc := NewXenditService(mockDB, mockXenditClient, mockUserClient)
	ctx := context.Background()

	event := models.OrderCreatedEvent{
		OrderID:     12345,
		UserID:      100,
		TotalAmount: 50000,
	}

	t.Run("success - creates invoice", func(t *testing.T) {
		mockUserClient.EXPECT().GetUserInfoByUserId(ctx, event.UserID).Return(&pb.GetUserInfoByUserIdResponse{
			Email: "user@test.com",
		}, nil)

		mockXenditClient.EXPECT().CreateInvoice(ctx, gomock.Any()).Return(&models.XenditInvoiceResponse{
			ID:         "inv-12345",
			InvoiceURL: "https://xendit.co/invoice/inv-12345",
			Status:     "PENDING",
			ExpireDate: time.Now().Add(24 * time.Hour),
		}, nil)

		mockDB.EXPECT().SavePayment(ctx, gomock.Any()).Return(nil)

		resp, err := svc.CreateInvoice(ctx, event)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "inv-12345", resp.ID)
	})

	t.Run("fails when user client is nil", func(t *testing.T) {
		svcNoClient := NewXenditService(mockDB, mockXenditClient, nil)

		_, err := svcNoClient.CreateInvoice(ctx, event)
		assert.Error(t, err)
	})

	t.Run("fails when gRPC call fails", func(t *testing.T) {
		mockUserClient.EXPECT().GetUserInfoByUserId(ctx, event.UserID).Return(nil, errors.New("grpc error"))

		_, err := svc.CreateInvoice(ctx, event)
		assert.Error(t, err)
	})

	t.Run("fails when Xendit API fails", func(t *testing.T) {
		mockUserClient.EXPECT().GetUserInfoByUserId(ctx, event.UserID).Return(&pb.GetUserInfoByUserIdResponse{
			Email: "user@test.com",
		}, nil)

		mockXenditClient.EXPECT().CreateInvoice(ctx, gomock.Any()).Return(nil, errors.New("xendit error"))

		_, err := svc.CreateInvoice(ctx, event)
		assert.Error(t, err)
	})

	t.Run("fails when save payment fails", func(t *testing.T) {
		mockUserClient.EXPECT().GetUserInfoByUserId(ctx, event.UserID).Return(&pb.GetUserInfoByUserIdResponse{
			Email: "user@test.com",
		}, nil)

		mockXenditClient.EXPECT().CreateInvoice(ctx, gomock.Any()).Return(&models.XenditInvoiceResponse{
			ID:         "inv-12345",
			InvoiceURL: "https://xendit.co/invoice/inv-12345",
			Status:     "PENDING",
			ExpireDate: time.Now().Add(24 * time.Hour),
		}, nil)

		mockDB.EXPECT().SavePayment(ctx, gomock.Any()).Return(errors.New("db error"))

		_, err := svc.CreateInvoice(ctx, event)
		assert.Error(t, err)
	})
}

func TestXenditService_CreateInvoiceFromPaymentRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockPaymentDatabase(ctrl)
	mockXenditClient := mocks.NewMockXenditClient(ctrl)
	mockUserClient := mocks.NewMockUserClientInterface(ctrl)

	svc := NewXenditService(mockDB, mockXenditClient, mockUserClient)
	ctx := context.Background()

	t.Run("success with existing email", func(t *testing.T) {
		pr := &models.PaymentRequest{
			ID:        1,
			OrderID:   12345,
			UserID:    100,
			Amount:    50000,
			UserEmail: "existing@test.com",
		}

		mockXenditClient.EXPECT().CreateInvoice(ctx, gomock.Any()).Return(&models.XenditInvoiceResponse{
			ID:         "inv-12345",
			InvoiceURL: "https://xendit.co/invoice/inv-12345",
			Status:     "PENDING",
		}, nil)

		mockDB.EXPECT().SavePayment(ctx, gomock.Any()).Return(nil)

		resp, err := svc.CreateInvoiceFromPaymentRequest(ctx, pr)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("success - fetches email via gRPC when not provided", func(t *testing.T) {
		pr := &models.PaymentRequest{
			ID:        1,
			OrderID:   12345,
			UserID:    100,
			Amount:    50000,
			UserEmail: "",
		}

		mockUserClient.EXPECT().GetUserInfoByUserId(ctx, pr.UserID).Return(&pb.GetUserInfoByUserIdResponse{
			Email: "fetched@test.com",
		}, nil)

		mockXenditClient.EXPECT().CreateInvoice(ctx, gomock.Any()).Return(&models.XenditInvoiceResponse{
			ID:         "inv-12345",
			InvoiceURL: "https://xendit.co/invoice/inv-12345",
			Status:     "PENDING",
		}, nil)

		mockDB.EXPECT().SavePayment(ctx, gomock.Any()).Return(nil)

		resp, err := svc.CreateInvoiceFromPaymentRequest(ctx, pr)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("fails when user client nil and no email", func(t *testing.T) {
		svcNoClient := NewXenditService(mockDB, mockXenditClient, nil)

		pr := &models.PaymentRequest{
			ID:        1,
			OrderID:   12345,
			UserID:    100,
			Amount:    50000,
			UserEmail: "",
		}

		_, err := svcNoClient.CreateInvoiceFromPaymentRequest(ctx, pr)
		assert.Error(t, err)
	})

	t.Run("fails when gRPC call fails", func(t *testing.T) {
		pr := &models.PaymentRequest{
			ID:        1,
			OrderID:   12345,
			UserID:    100,
			Amount:    50000,
			UserEmail: "",
		}

		mockUserClient.EXPECT().GetUserInfoByUserId(ctx, pr.UserID).Return(nil, errors.New("grpc error"))

		_, err := svc.CreateInvoiceFromPaymentRequest(ctx, pr)
		assert.Error(t, err)
	})
}

func TestXenditService_CheckInvoiceStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockPaymentDatabase(ctrl)
	mockXenditClient := mocks.NewMockXenditClient(ctrl)
	mockUserClient := mocks.NewMockUserClientInterface(ctrl)

	svc := NewXenditService(mockDB, mockXenditClient, mockUserClient)
	ctx := context.Background()

	t.Run("returns status successfully", func(t *testing.T) {
		externalID := "order-12345"
		mockXenditClient.EXPECT().CheckInvoiceStatus(ctx, externalID).Return("PAID", nil)

		status, err := svc.CheckInvoiceStatus(ctx, externalID)
		assert.NoError(t, err)
		assert.Equal(t, "PAID", status)
	})

	t.Run("returns error on failure", func(t *testing.T) {
		externalID := "order-12345"
		mockXenditClient.EXPECT().CheckInvoiceStatus(ctx, externalID).Return("", errors.New("api error"))

		_, err := svc.CheckInvoiceStatus(ctx, externalID)
		assert.Error(t, err)
	})
}
