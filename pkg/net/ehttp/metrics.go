package ehttp

import "github.com/prometheus/client_golang/prometheus"

var (
	namespace = "era"
	subsystem = "http"

	metricsHttpRequestCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "http_request_total",
		Help:      "total number of http request times",
	}, []string{
		"url",
	})

	metricsHttpRequestDurationHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "http_request_duration",
		Help:      "duration histogram of http request",
		Buckets:   []float64{10, 50, 100, 500, 1000, 10000, 50000},
	}, []string{
		"url",
	})
)

func init() {
	prometheus.MustRegister(metricsHttpRequestCounter, metricsHttpRequestDurationHistogram)
}
