package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// XenditWebhookProcessed 콜백 처리 결과 (비즈니스 관측).
var XenditWebhookProcessed = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "commerce",
		Subsystem: "payment",
		Name:      "xendit_webhook_processed_total",
		Help:      "Xendit webhook invocations by outcome",
	},
	[]string{"outcome"},
)
