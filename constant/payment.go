package constant

const (
	PaymentStatusPaid    = "PAID"
	PaymentStatusFailed  = "FAILED"
	PaymentStatusPending = "PENDING"
	PaymentStatusExpired = "EXPIRED"
)

const (
	KafkaTopicPaymentSuccess = "payment.success"
	KafkaTopicOrderCreated   = "order.created"
)

// MaxRetryPublish payment.success Kafka 발행 최대 재시도 횟수
const MaxRetryPublish = 3
