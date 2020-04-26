package redis

import "github.com/prometheus/client_golang/prometheus"

var (
	namespace = "era"
	subsystem = "redis"

	metricsRedisCmdCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "cmd_exec_total",
		Help:      "total number of cmd execution times",
	}, []string{
		"cmd",
	})

	metricsRedisDurationHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "cmd_exec_duration",
		Help:      "duration histogram of cmd execution",
		Buckets:   []float64{1, 10, 50, 100, 500, 1000},
	}, []string{
		"cmd",
	})

	metricsRedisStatsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "performance_statistics",
		Help:      "performance statistics",
	}, []string{
		"stats",
	})
)

func init() {
	prometheus.MustRegister(metricsRedisCmdCounter, metricsRedisDurationHistogram, metricsRedisStatsGauge)
}
