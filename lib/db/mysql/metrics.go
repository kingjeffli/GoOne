package mysql

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var mysqlDurationBuckets = []float64{
	0.0005,
	0.001,
	0.005,
	0.01,
	0.025,
	0.05,
	0.1,
	0.25,
	0.5,
	1,
	2.5,
}

var (
	mysqlOperationsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "goone_mysql_operations_total",
		Help: "Total MySQL operations by instance, operation, and result.",
	}, []string{"instance", "operation", "result"})
	mysqlOperationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "goone_mysql_operation_duration_seconds",
		Help:    "Latency distribution of MySQL operations.",
		Buckets: mysqlDurationBuckets,
	}, []string{"instance", "operation"})
	mysqlOperationErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "goone_mysql_operation_errors_total",
		Help: "Total MySQL operation errors.",
	}, []string{"instance", "operation"})
	mysqlOperationTimeouts = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "goone_mysql_operation_timeouts_total",
		Help: "Total MySQL operation timeouts.",
	}, []string{"instance", "operation"})
	mysqlOperationsInFlight = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "goone_mysql_operations_in_flight",
		Help: "Current in-flight MySQL operations.",
	}, []string{"instance", "operation"})

	mysqlPoolConnectionsDesc = prometheus.NewDesc(
		"goone_mysql_pool_connections",
		"Current MySQL pool connections by instance and state.",
		[]string{"instance", "state"},
		nil,
	)
	mysqlPoolWaitCountDesc = prometheus.NewDesc(
		"goone_mysql_pool_wait_count_total",
		"Total MySQL pool waits by instance.",
		[]string{"instance"},
		nil,
	)
	mysqlPoolWaitDurationDesc = prometheus.NewDesc(
		"goone_mysql_pool_wait_duration_seconds_total",
		"Total MySQL pool wait duration by instance.",
		[]string{"instance"},
		nil,
	)
)

var (
	mysqlCollectorOnce sync.Once
	mysqlDBRegistry    sync.Map // map[uint32]*sql.DB
)

type mysqlPoolCollector struct{}

func registerMySQLDB(instID uint32, db *sql.DB) {
	if db == nil {
		return
	}
	mysqlCollectorOnce.Do(func() {
		prometheus.MustRegister(mysqlPoolCollector{})
	})
	mysqlDBRegistry.Store(instID, db)
}

func (mysqlPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- mysqlPoolConnectionsDesc
	ch <- mysqlPoolWaitCountDesc
	ch <- mysqlPoolWaitDurationDesc
}

func (mysqlPoolCollector) Collect(ch chan<- prometheus.Metric) {
	mysqlDBRegistry.Range(func(key, value any) bool {
		instID, ok := key.(uint32)
		if !ok {
			return true
		}
		db, ok := value.(*sql.DB)
		if !ok || db == nil {
			return true
		}

		stats := db.Stats()
		instance := fmt.Sprintf("%d", instID)
		ch <- prometheus.MustNewConstMetric(mysqlPoolConnectionsDesc, prometheus.GaugeValue, float64(stats.OpenConnections), instance, "open")
		ch <- prometheus.MustNewConstMetric(mysqlPoolConnectionsDesc, prometheus.GaugeValue, float64(stats.InUse), instance, "in_use")
		ch <- prometheus.MustNewConstMetric(mysqlPoolConnectionsDesc, prometheus.GaugeValue, float64(stats.Idle), instance, "idle")
		ch <- prometheus.MustNewConstMetric(mysqlPoolConnectionsDesc, prometheus.GaugeValue, float64(stats.MaxOpenConnections), instance, "max_open")
		ch <- prometheus.MustNewConstMetric(mysqlPoolWaitCountDesc, prometheus.CounterValue, float64(stats.WaitCount), instance)
		ch <- prometheus.MustNewConstMetric(mysqlPoolWaitDurationDesc, prometheus.CounterValue, stats.WaitDuration.Seconds(), instance)
		return true
	})
}

func beginMySQLObserve(instID uint32, query string, fallback string) func(err error) {
	instanceLabel := fmt.Sprintf("%d", instID)
	operationLabel := normalizeSQLOperation(query, fallback)
	mysqlOperationsInFlight.WithLabelValues(instanceLabel, operationLabel).Inc()
	start := time.Now()
	return func(err error) {
		mysqlOperationsInFlight.WithLabelValues(instanceLabel, operationLabel).Dec()
		mysqlOperationsTotal.WithLabelValues(instanceLabel, operationLabel, mysqlResultLabel(err)).Inc()
		mysqlOperationDuration.WithLabelValues(instanceLabel, operationLabel).Observe(time.Since(start).Seconds())
		if err != nil {
			mysqlOperationErrors.WithLabelValues(instanceLabel, operationLabel).Inc()
			if mysqlIsTimeoutErr(err) {
				mysqlOperationTimeouts.WithLabelValues(instanceLabel, operationLabel).Inc()
			}
		}
	}
}

func normalizeSQLOperation(query, fallback string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		query = fallback
	}
	if query == "" {
		return "UNKNOWN"
	}
	fields := strings.Fields(query)
	if len(fields) == 0 {
		return "UNKNOWN"
	}
	return strings.ToUpper(fields[0])
}

func mysqlResultLabel(err error) string {
	if err == nil {
		return "ok"
	}
	if mysqlIsTimeoutErr(err) {
		return "timeout"
	}
	return "error"
}

func mysqlIsTimeoutErr(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded")
}
