package models

type XenditWebhookPayload struct {
	ExternalID string `json:"external_id"`
	Status     string `json:"status"`
}
