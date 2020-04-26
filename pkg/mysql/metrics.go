package mysql

import "github.com/prometheus/client_golang/prometheus"

var (
	namespace = "era"
	subsystem = "mysql"

	metricsMysqlQueryCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "query_exec_total",
		Help:      "total number of query execution times",
	}, []string{
		"db", "query",
	})

	metricsMysqlDurationHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "query_exec_duration",
		Help:      "duration histogram of query execution",
		Buckets:   []float64{1, 10, 50, 100, 500, 1000, 10000, 50000},
	}, []string{
		"db", "query",
	})

	metricsMysqlStatsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "db_statistics",
		Help:      "db performance statistics",
	}, []string{
		"db", "stats",
	})
)

func init() {
	prometheus.MustRegister(metricsMysqlQueryCounter, metricsMysqlDurationHistogram, metricsMysqlStatsGauge)
}
