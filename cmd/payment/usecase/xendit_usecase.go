package usecase

import (
	"context"
	"paymentfc/cmd/payment/service"
	"paymentfc/models"
)

type XenditUsecase interface {
	CreateInvoice(ctx context.Context, param models.OrderCreatedEvent) (*models.XenditInvoiceResponse, error)
	CheckInvoiceStatus(ctx context.Context, externalID string) (string, error)
}

type xenditUsecase struct {
	xenditService service.XenditService
}

func NewXenditUsecase(xenditService service.XenditService) XenditUsecase {
	return &xenditUsecase{xenditService: xenditService}
}

func (u *xenditUsecase) CreateInvoice(ctx context.Context, param models.OrderCreatedEvent) (*models.XenditInvoiceResponse, error) {
	return u.xenditService.CreateInvoice(ctx, param)
}

func (u *xenditUsecase) CheckInvoiceStatus(ctx context.Context, externalID string) (string, error) {
	return u.xenditService.CheckInvoiceStatus(ctx, externalID)
}
