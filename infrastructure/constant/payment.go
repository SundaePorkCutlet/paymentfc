package constant

const (
	PaymentStatusPaid    = "PAID"
	PaymentStatusFailed  = "FAILED"
	PaymentStatusPending = "PENDING"
)

const (
	KafkaTopicPaymentSuccess = "payment.success"
	KafkaTopicOrderCreated   = "order.created"
)
