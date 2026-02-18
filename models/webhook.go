package models

type XenditWebhookPayload struct {
	ExternalID string  `json:"external_id"`
	Status     string  `json:"status"`
	Amount     float64 `json:"amount"` // 웹훅에서 오면 총액 검증에 사용
}
