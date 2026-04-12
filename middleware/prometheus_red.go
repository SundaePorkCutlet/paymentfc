package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "commerce",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total HTTP requests by route template, method, status",
		},
		[]string{"service", "method", "path", "status"},
	)
	httpDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "commerce",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"service", "method", "path"},
	)
)

// PrometheusRED registers request count and latency histogram per route.
func PrometheusRED(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}
		method := c.Request.Method

		httpRequests.WithLabelValues(service, method, path, status).Inc()
		httpDuration.WithLabelValues(service, method, path).Observe(time.Since(start).Seconds())
	}
}
