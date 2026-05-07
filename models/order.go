package models

type OrderCreatedEvent struct {
	OrderID         int64   `json:"order_id"`
	UserID          int64   `json:"user_id"`
	TotalAmount     float64 `json:"total_amount"`
	PaymentMethod   string  `json:"payment_method"`
	ShippingAddress string  `json:"shipping_address"`
}

type ProductItem struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

type StockReservationEvent struct {
	SchemaVersion int           `json:"schema_version"`
	OrderID       int64         `json:"order_id"`
	UserID        int64         `json:"user_id"`
	TotalAmount   float64       `json:"total_amount"`
	Products      []ProductItem `json:"products"`
	EventTime     string        `json:"event_time"`
}
