package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"paymentfc/infrastructure/log"
	"paymentfc/models"
)

type XenditClient interface {
	CreateInvoice(ctx context.Context, request models.XenditInvoiceRequest) (*models.XenditInvoiceResponse, error)
	CheckInvoiceStatus(ctx context.Context, externalID string) (string, error)
}

type xenditClient struct {
	apiKey string
}

func NewXenditClient(apiKey string) XenditClient {
	return &xenditClient{apiKey: apiKey}
}

func (x *xenditClient) CreateInvoice(ctx context.Context, request models.XenditInvoiceRequest) (*models.XenditInvoiceResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to marshal invoice request")
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.xendit.co/v2/invoices", bytes.NewBuffer(body))
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to create HTTP request")
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(x.apiKey, "")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to call Xendit API")
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Logger.Error().Msgf("Xendit API returned status: %d", resp.StatusCode)
		return nil, fmt.Errorf("xendit API returned status: %d", resp.StatusCode)
	}

	var invoiceResponse models.XenditInvoiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&invoiceResponse); err != nil {
		log.Logger.Error().Err(err).Msg("Failed to decode Xendit response")
		return nil, err
	}

	return &invoiceResponse, nil
}

func (x *xenditClient) CheckInvoiceStatus(ctx context.Context, externalID string) (string, error) {
	url := fmt.Sprintf("https://api.xendit.co/v2/invoices?external_id=%s", externalID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	httpReq.SetBasicAuth(x.apiKey, "")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to call Xendit API")
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Logger.Error().Msgf("Xendit API returned status: %d", resp.StatusCode)
		return "", fmt.Errorf("xendit API returned status: %d", resp.StatusCode)
	}

	var invoiceResponse []models.XenditInvoiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&invoiceResponse); err != nil {
		log.Logger.Error().Err(err).Msg("Failed to decode Xendit response")
		return "", err
	}

	if len(invoiceResponse) == 0 {
		return "", fmt.Errorf("no invoice found for external_id: %s", externalID)
	}

	return invoiceResponse[0].Status, nil
}
