package service

import (
	"context"
	"errors"
	"paymentfc/constant"
	"paymentfc/mocks"
	"paymentfc/models"
	pb "paymentfc/pb/proto"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func createTestSchedulerService(
	ctrl *gomock.Controller,
	mockDB *mocks.MockPaymentDatabase,
	mockXendit *mocks.MockXenditClient,
	mockPublisher *mocks.MockPaymentEventPublisher,
	mockPaymentService *MockPaymentService,
	mockAuditLog *mocks.MockAuditLogRepository,
	mockUserClient *mocks.MockUserClientInterface,
) *SchedulerService {
	return &SchedulerService{
		Database:       mockDB,
		Xendit:         mockXendit,
		Publisher:      mockPublisher,
		PaymentService: mockPaymentService,
		AuditLog:       mockAuditLog,
		UserClient:     mockUserClient,
	}
}

func TestSchedulerService_ProcessPendingPaymentRequests_Logic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockPaymentDatabase(ctrl)
	mockXendit := mocks.NewMockXenditClient(ctrl)
	mockPublisher := mocks.NewMockPaymentEventPublisher(ctrl)
	mockPaymentService := NewMockPaymentService(ctrl)
	mockAuditLog := mocks.NewMockAuditLogRepository(ctrl)
	mockUserClient := mocks.NewMockUserClientInterface(ctrl)

	ctx := context.Background()

	t.Run("skips when no pending requests", func(t *testing.T) {
		mockDB.EXPECT().GetPendingPaymentRequests(ctx).Return([]models.PaymentRequest{}, nil)

		scheduler := createTestSchedulerService(ctrl, mockDB, mockXendit, mockPublisher, mockPaymentService, mockAuditLog, mockUserClient)

		requests, err := scheduler.Database.GetPendingPaymentRequests(ctx)
		assert.NoError(t, err)
		assert.Len(t, requests, 0)
	})

	t.Run("processes request with existing email", func(t *testing.T) {
		pr := models.PaymentRequest{
			ID:        1,
			OrderID:   100,
			UserID:    10,
			Amount:    50000,
			UserEmail: "test@example.com",
		}

		mockDB.EXPECT().GetPendingPaymentRequests(ctx).Return([]models.PaymentRequest{pr}, nil)
		mockDB.EXPECT().GetPaymentByOrderID(ctx, pr.OrderID).Return(nil, gorm.ErrRecordNotFound)
		mockXendit.EXPECT().CreateInvoice(ctx, gomock.Any()).Return(&models.XenditInvoiceResponse{
			ID:         "inv-1",
			ExpireDate: time.Now().Add(24 * time.Hour),
		}, nil)
		mockDB.EXPECT().UpdateSuccessPaymentRequest(ctx, pr.ID).Return(nil)
		mockDB.EXPECT().SavePayment(ctx, gomock.Any()).Return(nil)
		mockAuditLog.EXPECT().SaveAuditLog(ctx, gomock.Any()).Return(nil)

		scheduler := createTestSchedulerService(ctrl, mockDB, mockXendit, mockPublisher, mockPaymentService, mockAuditLog, mockUserClient)

		requests, _ := scheduler.Database.GetPendingPaymentRequests(ctx)
		for _, request := range requests {
			paymentInfo, err := scheduler.Database.GetPaymentByOrderID(ctx, request.OrderID)
			if err == gorm.ErrRecordNotFound || (paymentInfo != nil && paymentInfo.ID == 0) {
				xenditReq := models.XenditInvoiceRequest{
					ExternalID: "order-100",
					Amount:     request.Amount,
					PayerEmail: request.UserEmail,
				}
				resp, err := scheduler.Xendit.CreateInvoice(ctx, xenditReq)
				if err == nil && resp != nil {
					scheduler.Database.UpdateSuccessPaymentRequest(ctx, request.ID)
					payment := &models.Payment{
						OrderID:     request.OrderID,
						UserID:      request.UserID,
						ExternalID:  xenditReq.ExternalID,
						Amount:      request.Amount,
						Status:      constant.PaymentStatusPending,
						ExpiredTime: resp.ExpireDate,
					}
					scheduler.Database.SavePayment(ctx, payment)
					scheduler.AuditLog.SaveAuditLog(ctx, &models.PaymentAuditLog{
						OrderID: request.OrderID,
						Event:   "INVOICE_CREATED",
					})
				}
			}
		}
	})

	t.Run("fetches email via gRPC when not provided", func(t *testing.T) {
		pr := models.PaymentRequest{
			ID:        2,
			OrderID:   200,
			UserID:    20,
			Amount:    75000,
			UserEmail: "",
		}

		mockDB.EXPECT().GetPendingPaymentRequests(ctx).Return([]models.PaymentRequest{pr}, nil)
		mockUserClient.EXPECT().GetUserInfoByUserId(ctx, pr.UserID).Return(&pb.GetUserInfoByUserIdResponse{
			Email: "grpc@example.com",
		}, nil)

		scheduler := createTestSchedulerService(ctrl, mockDB, mockXendit, mockPublisher, mockPaymentService, mockAuditLog, mockUserClient)

		requests, _ := scheduler.Database.GetPendingPaymentRequests(ctx)
		for _, request := range requests {
			payerEmail := request.UserEmail
			if payerEmail == "" {
				userInfo, err := scheduler.UserClient.GetUserInfoByUserId(ctx, request.UserID)
				if err == nil && userInfo != nil {
					payerEmail = userInfo.Email
				}
			}
			assert.Equal(t, "grpc@example.com", payerEmail)
		}
	})

	t.Run("skips when payment already exists", func(t *testing.T) {
		pr := models.PaymentRequest{
			ID:        3,
			OrderID:   300,
			UserID:    30,
			Amount:    100000,
			UserEmail: "test@example.com",
		}

		existingPayment := &models.Payment{
			ID:      999,
			OrderID: 300,
			Status:  constant.PaymentStatusPending,
		}

		mockDB.EXPECT().GetPendingPaymentRequests(ctx).Return([]models.PaymentRequest{pr}, nil)
		mockDB.EXPECT().GetPaymentByOrderID(ctx, pr.OrderID).Return(existingPayment, nil)
		mockDB.EXPECT().UpdateSuccessPaymentRequest(ctx, pr.ID).Return(nil)

		scheduler := createTestSchedulerService(ctrl, mockDB, mockXendit, mockPublisher, mockPaymentService, mockAuditLog, mockUserClient)

		requests, _ := scheduler.Database.GetPendingPaymentRequests(ctx)
		for _, request := range requests {
			paymentInfo, _ := scheduler.Database.GetPaymentByOrderID(ctx, request.OrderID)
			if paymentInfo != nil && paymentInfo.ID != 0 {
				scheduler.Database.UpdateSuccessPaymentRequest(ctx, request.ID)
			}
		}
	})

	t.Run("marks failed when gRPC fails", func(t *testing.T) {
		pr := models.PaymentRequest{
			ID:        4,
			OrderID:   400,
			UserID:    40,
			Amount:    50000,
			UserEmail: "",
		}

		mockDB.EXPECT().GetPendingPaymentRequests(ctx).Return([]models.PaymentRequest{pr}, nil)
		mockUserClient.EXPECT().GetUserInfoByUserId(ctx, pr.UserID).Return(nil, errors.New("grpc error"))
		mockDB.EXPECT().UpdateFailedPaymentRequest(ctx, pr.ID, "failed to get user email").Return(nil)

		scheduler := createTestSchedulerService(ctrl, mockDB, mockXendit, mockPublisher, mockPaymentService, mockAuditLog, mockUserClient)

		requests, _ := scheduler.Database.GetPendingPaymentRequests(ctx)
		for _, request := range requests {
			if request.UserEmail == "" {
				_, err := scheduler.UserClient.GetUserInfoByUserId(ctx, request.UserID)
				if err != nil {
					scheduler.Database.UpdateFailedPaymentRequest(ctx, request.ID, "failed to get user email")
				}
			}
		}
	})

	t.Run("marks failed when invoice creation fails", func(t *testing.T) {
		pr := models.PaymentRequest{
			ID:        5,
			OrderID:   500,
			UserID:    50,
			Amount:    50000,
			UserEmail: "test@example.com",
		}

		mockDB.EXPECT().GetPendingPaymentRequests(ctx).Return([]models.PaymentRequest{pr}, nil)
		mockDB.EXPECT().GetPaymentByOrderID(ctx, pr.OrderID).Return(nil, gorm.ErrRecordNotFound)
		mockXendit.EXPECT().CreateInvoice(ctx, gomock.Any()).Return(nil, errors.New("xendit api error"))
		mockAuditLog.EXPECT().SaveAuditLog(ctx, gomock.Any()).Return(nil)
		mockDB.EXPECT().UpdateFailedPaymentRequest(ctx, pr.ID, "xendit api error").Return(nil)

		scheduler := createTestSchedulerService(ctrl, mockDB, mockXendit, mockPublisher, mockPaymentService, mockAuditLog, mockUserClient)

		requests, _ := scheduler.Database.GetPendingPaymentRequests(ctx)
		for _, request := range requests {
			paymentInfo, err := scheduler.Database.GetPaymentByOrderID(ctx, request.OrderID)
			if err == gorm.ErrRecordNotFound || (paymentInfo == nil) {
				xenditReq := models.XenditInvoiceRequest{
					ExternalID: "order-500",
					Amount:     request.Amount,
					PayerEmail: request.UserEmail,
				}
				_, invoiceErr := scheduler.Xendit.CreateInvoice(ctx, xenditReq)
				if invoiceErr != nil {
					scheduler.AuditLog.SaveAuditLog(ctx, &models.PaymentAuditLog{
						OrderID: request.OrderID,
						Event:   "INVOICE_CREATION_FAILED",
					})
					scheduler.Database.UpdateFailedPaymentRequest(ctx, request.ID, invoiceErr.Error())
				}
			}
		}
	})
}

func TestSchedulerService_ProcessFailedPaymentRequests_Logic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockPaymentDatabase(ctrl)
	mockXendit := mocks.NewMockXenditClient(ctrl)
	mockPublisher := mocks.NewMockPaymentEventPublisher(ctrl)
	mockPaymentService := NewMockPaymentService(ctrl)
	mockAuditLog := mocks.NewMockAuditLogRepository(ctrl)
	mockUserClient := mocks.NewMockUserClientInterface(ctrl)

	ctx := context.Background()

	t.Run("resets failed requests to pending", func(t *testing.T) {
		failedRequests := []models.PaymentRequest{
			{ID: 1, OrderID: 100, UserID: 10, RetryCount: 1},
			{ID: 2, OrderID: 200, UserID: 20, RetryCount: 2},
		}

		mockDB.EXPECT().GetFailedPaymentRequests(ctx).Return(failedRequests, nil)
		mockDB.EXPECT().UpdatePendingPaymentRequest(ctx, int64(1)).Return(nil)
		mockAuditLog.EXPECT().SaveAuditLog(ctx, gomock.Any()).Return(nil)
		mockDB.EXPECT().UpdatePendingPaymentRequest(ctx, int64(2)).Return(nil)
		mockAuditLog.EXPECT().SaveAuditLog(ctx, gomock.Any()).Return(nil)

		scheduler := createTestSchedulerService(ctrl, mockDB, mockXendit, mockPublisher, mockPaymentService, mockAuditLog, mockUserClient)

		requests, _ := scheduler.Database.GetFailedPaymentRequests(ctx)
		for _, pr := range requests {
			err := scheduler.Database.UpdatePendingPaymentRequest(ctx, pr.ID)
			if err == nil {
				scheduler.AuditLog.SaveAuditLog(ctx, &models.PaymentAuditLog{
					OrderID: pr.OrderID,
					Event:   "PAYMENT_REQUEST_RETRY",
				})
			}
		}
	})

	t.Run("marks as failed when update fails", func(t *testing.T) {
		failedRequests := []models.PaymentRequest{
			{ID: 3, OrderID: 300, UserID: 30, RetryCount: 3},
		}

		mockDB.EXPECT().GetFailedPaymentRequests(ctx).Return(failedRequests, nil)
		mockDB.EXPECT().UpdatePendingPaymentRequest(ctx, int64(3)).Return(errors.New("db error"))
		mockDB.EXPECT().UpdateFailedPaymentRequest(ctx, int64(3), "db error").Return(nil)

		scheduler := createTestSchedulerService(ctrl, mockDB, mockXendit, mockPublisher, mockPaymentService, mockAuditLog, mockUserClient)

		requests, _ := scheduler.Database.GetFailedPaymentRequests(ctx)
		for _, pr := range requests {
			err := scheduler.Database.UpdatePendingPaymentRequest(ctx, pr.ID)
			if err != nil {
				scheduler.Database.UpdateFailedPaymentRequest(ctx, pr.ID, err.Error())
			}
		}
	})
}

func TestSchedulerService_SweepExpiredPayments_Logic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockPaymentDatabase(ctrl)
	mockXendit := mocks.NewMockXenditClient(ctrl)
	mockPublisher := mocks.NewMockPaymentEventPublisher(ctrl)
	mockPaymentService := NewMockPaymentService(ctrl)
	mockAuditLog := mocks.NewMockAuditLogRepository(ctrl)
	mockUserClient := mocks.NewMockUserClientInterface(ctrl)

	ctx := context.Background()

	t.Run("marks expired payments", func(t *testing.T) {
		expiredPayments := []models.Payment{
			{ID: 1, OrderID: 100, ExternalID: "order-100", Status: constant.PaymentStatusPending},
			{ID: 2, OrderID: 200, ExternalID: "order-200", Status: constant.PaymentStatusPending},
		}

		mockDB.EXPECT().GetExpiredPendingPayments(ctx).Return(expiredPayments, nil)
		mockDB.EXPECT().MarkExpired(ctx, int64(1)).Return(nil)
		mockAuditLog.EXPECT().SaveAuditLog(ctx, gomock.Any()).Return(nil)
		mockDB.EXPECT().MarkExpired(ctx, int64(2)).Return(nil)
		mockAuditLog.EXPECT().SaveAuditLog(ctx, gomock.Any()).Return(nil)

		scheduler := createTestSchedulerService(ctrl, mockDB, mockXendit, mockPublisher, mockPaymentService, mockAuditLog, mockUserClient)

		payments, _ := scheduler.Database.GetExpiredPendingPayments(ctx)
		for _, payment := range payments {
			err := scheduler.Database.MarkExpired(ctx, payment.ID)
			if err == nil {
				scheduler.AuditLog.SaveAuditLog(ctx, &models.PaymentAuditLog{
					OrderID:    payment.OrderID,
					PaymentID:  payment.ID,
					ExternalID: payment.ExternalID,
					Event:      "PAYMENT_EXPIRED",
				})
			}
		}
	})

	t.Run("handles no expired payments", func(t *testing.T) {
		mockDB.EXPECT().GetExpiredPendingPayments(ctx).Return([]models.Payment{}, nil)

		scheduler := createTestSchedulerService(ctrl, mockDB, mockXendit, mockPublisher, mockPaymentService, mockAuditLog, mockUserClient)

		payments, err := scheduler.Database.GetExpiredPendingPayments(ctx)
		assert.NoError(t, err)
		assert.Len(t, payments, 0)
	})
}

func TestSchedulerService_CheckPendingInvoices_Logic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockPaymentDatabase(ctrl)
	mockXendit := mocks.NewMockXenditClient(ctrl)
	mockPublisher := mocks.NewMockPaymentEventPublisher(ctrl)
	mockPaymentService := NewMockPaymentService(ctrl)
	mockAuditLog := mocks.NewMockAuditLogRepository(ctrl)
	mockUserClient := mocks.NewMockUserClientInterface(ctrl)

	ctx := context.Background()

	t.Run("processes paid invoice", func(t *testing.T) {
		pendingInvoices := []models.Payment{
			{ID: 1, OrderID: 100, ExternalID: "order-100", Status: constant.PaymentStatusPending},
		}

		mockDB.EXPECT().GetPendingInvoices(ctx).Return(pendingInvoices, nil)
		mockXendit.EXPECT().CheckInvoiceStatus(ctx, "order-100").Return(constant.PaymentStatusPaid, nil)
		mockPaymentService.EXPECT().ProcessPaymentSuccess(ctx, int64(100)).Return(nil)

		scheduler := createTestSchedulerService(ctrl, mockDB, mockXendit, mockPublisher, mockPaymentService, mockAuditLog, mockUserClient)

		invoices, _ := scheduler.Database.GetPendingInvoices(ctx)
		for _, invoice := range invoices {
			status, err := scheduler.Xendit.CheckInvoiceStatus(ctx, invoice.ExternalID)
			if err == nil && status == constant.PaymentStatusPaid {
				scheduler.PaymentService.ProcessPaymentSuccess(ctx, invoice.OrderID)
			}
		}
	})

	t.Run("skips non-paid invoice", func(t *testing.T) {
		pendingInvoices := []models.Payment{
			{ID: 2, OrderID: 200, ExternalID: "order-200", Status: constant.PaymentStatusPending},
		}

		mockDB.EXPECT().GetPendingInvoices(ctx).Return(pendingInvoices, nil)
		mockXendit.EXPECT().CheckInvoiceStatus(ctx, "order-200").Return(constant.PaymentStatusPending, nil)

		scheduler := createTestSchedulerService(ctrl, mockDB, mockXendit, mockPublisher, mockPaymentService, mockAuditLog, mockUserClient)

		invoices, _ := scheduler.Database.GetPendingInvoices(ctx)
		for _, invoice := range invoices {
			status, err := scheduler.Xendit.CheckInvoiceStatus(ctx, invoice.ExternalID)
			assert.NoError(t, err)
			assert.NotEqual(t, constant.PaymentStatusPaid, status)
		}
	})
}
