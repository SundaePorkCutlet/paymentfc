package service

import (
	"context"
	"fmt"
	"paymentfc/cmd/payment/repository"
	"paymentfc/infrastructure/constant"
	"paymentfc/infrastructure/log"
	"paymentfc/models"
	"time"
)

type SchedulerService struct {
	Database       repository.PaymentDatabase
	Xendit         repository.XenditClient
	Publisher      repository.PaymentEventPublisher
	PaymentService PaymentService
}

func (s *SchedulerService) StartCheckPendingInvoices() {
	ticker := time.NewTicker(10 * time.Minute)

	go func() {
		for range ticker.C {
			pendingInvoices, err := s.Database.GetPendingInvoices(context.Background())
			if err != nil {
				log.Logger.Error().Err(err).Msg("Failed to get pending invoices")
				continue
			}
			for _, pendingInvoice := range pendingInvoices {
				invoiceStatus, err := s.Xendit.CheckInvoiceStatus(context.Background(), pendingInvoice.ExternalID)
				if err != nil {
					log.Logger.Error().Err(err).Msgf("Failed to check invoice status for external_id: %s", pendingInvoice.ExternalID)
					continue
				}
				if invoiceStatus == constant.PaymentStatusPaid {
					err = s.PaymentService.ProcessPaymentSuccess(context.Background(), pendingInvoice.OrderID)
					if err != nil {
						log.Logger.Error().Err(err).Msgf("Failed to process payment success for order_id: %d", pendingInvoice.OrderID)
						continue
					}
				}
			}
		}
	}()
}

// StartProcessPaymentRequests 주기적으로 PENDING payment_requests를 읽어 인보이스 생성 (배치). 강의 방식: 스케줄러 안에서 직접 DB·Xendit 호출.
func (s *SchedulerService) StartProcessPaymentRequests() {
	go func() {
		for {
			ctx := context.Background()

			// get pending payment requests
			paymentRequests, err := s.Database.GetPendingPaymentRequests(ctx)
			if err != nil {
				log.Logger.Error().Err(err).Msg("s.Database.GetPendingPaymentRequests() got error")
				time.Sleep(5 * time.Second) // DB 이슈 시 잠시 대기 후 재시도
				continue
			}

			// looping list of pending payment requests
			for _, pr := range paymentRequests {
				log.Logger.Debug().Int64("order_id", pr.OrderID).Msg("Processing payment request")

				payerEmail := pr.UserEmail
				if payerEmail == "" {
					payerEmail = fmt.Sprintf("user%d@test.com", pr.UserID)
				}
				xenditReq := models.XenditInvoiceRequest{
					ExternalID:  fmt.Sprintf("order-%d", pr.OrderID),
					Amount:     pr.Amount,
					Description: fmt.Sprintf("[FC] Pembayaran Order %d", pr.OrderID),
					PayerEmail: payerEmail,
				}

				_, err := s.Xendit.CreateInvoice(ctx, xenditReq)
				if err != nil {
					log.Logger.Error().Err(err).Int64("order_id", pr.OrderID).Msg("Failed to create invoice")
					if updateErr := s.Database.UpdateFailedPaymentRequest(ctx, pr.ID, err.Error()); updateErr != nil {
						log.Logger.Error().Err(updateErr).Int64("payment_request_id", pr.ID).Msg("Failed to update payment_request as failed")
					}
					continue
				}

				// 인보이스 생성 성공: payments 테이블에 저장 + payment_requests 상태 업데이트
				payment := &models.Payment{
					OrderID:    pr.OrderID,
					UserID:     pr.UserID,
					ExternalID: xenditReq.ExternalID,
					Amount:     pr.Amount,
					Status:     constant.PaymentStatusPending,
					CreateTime: time.Now(),
				}
				if err := s.Database.SavePayment(ctx, payment); err != nil {
					log.Logger.Error().Err(err).Int64("order_id", pr.OrderID).Msg("Failed to save payment")
					continue
				}
				if err := s.Database.UpdateSuccessPaymentRequest(ctx, pr.ID); err != nil {
					log.Logger.Error().Err(err).Int64("payment_request_id", pr.ID).Msg("Failed to update payment_request as success")
				}
			}

			time.Sleep(1 * time.Minute) // 정상 처리 후 1분 대기
		}
	}()
}
