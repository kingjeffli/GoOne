package orm

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/go-xorm/xorm"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var xormDurationBuckets = []float64{
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
	xormPingTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "goone_xorm_ping_total",
		Help: "Total xorm Ping attempts by db name and result.",
	}, []string{"db", "result"})
	xormPingDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "goone_xorm_ping_duration_seconds",
		Help:    "Latency distribution of xorm Ping calls.",
		Buckets: xormDurationBuckets,
	}, []string{"db"})
	xormPingErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "goone_xorm_ping_errors_total",
		Help: "Total xorm Ping errors by db name.",
	}, []string{"db"})
	xormPingTimeouts = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "goone_xorm_ping_timeouts_total",
		Help: "Total xorm Ping timeouts by db name.",
	}, []string{"db"})

	xormPoolConnectionsDesc = prometheus.NewDesc(
		"goone_xorm_pool_connections",
		"Current xorm-backed MySQL pool connections by db, role, and state.",
		[]string{"db", "role", "state"},
		nil,
	)
	xormPoolWaitCountDesc = prometheus.NewDesc(
		"goone_xorm_pool_wait_count_total",
		"Total xorm-backed MySQL pool waits by db and role.",
		[]string{"db", "role"},
		nil,
	)
	xormPoolWaitDurationDesc = prometheus.NewDesc(
		"goone_xorm_pool_wait_duration_seconds_total",
		"Total xorm-backed MySQL pool wait duration by db and role.",
		[]string{"db", "role"},
		nil,
	)
)

var (
	xormCollectorOnce sync.Once
	xormRegistry      sync.Map // map[string]*OrmSql
)

type xormPoolCollector struct{}

func registerOrmMetrics(name string, orm *OrmSql) {
	if orm == nil || strings.TrimSpace(name) == "" {
		return
	}
	xormCollectorOnce.Do(func() {
		prometheus.MustRegister(xormPoolCollector{})
	})
	xormRegistry.Store(name, orm)
}

func (xormPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- xormPoolConnectionsDesc
	ch <- xormPoolWaitCountDesc
	ch <- xormPoolWaitDurationDesc
}

func (xormPoolCollector) Collect(ch chan<- prometheus.Metric) {
	xormRegistry.Range(func(key, value any) bool {
		name, ok := key.(string)
		if !ok || strings.TrimSpace(name) == "" {
			return true
		}
		orm, ok := value.(*OrmSql)
		if !ok || orm == nil || orm.Engine == nil {
			return true
		}

		collectXormEngineStats(ch, name, "master", orm.Engine.Master())
		for idx, slave := range orm.Engine.Slaves() {
			role := "slave"
			if len(orm.Engine.Slaves()) > 1 {
				role = fmt.Sprintf("slave_%d", idx)
			}
			collectXormEngineStats(ch, name, role, slave)
		}
		return true
	})
}

func collectXormEngineStats(ch chan<- prometheus.Metric, name, role string, engine *xorm.Engine) {
	if engine == nil || engine.DB() == nil || engine.DB().DB == nil {
		return
	}
	stats := engine.DB().Stats()
	ch <- prometheus.MustNewConstMetric(xormPoolConnectionsDesc, prometheus.GaugeValue, float64(stats.OpenConnections), name, role, "open")
	ch <- prometheus.MustNewConstMetric(xormPoolConnectionsDesc, prometheus.GaugeValue, float64(stats.InUse), name, role, "in_use")
	ch <- prometheus.MustNewConstMetric(xormPoolConnectionsDesc, prometheus.GaugeValue, float64(stats.Idle), name, role, "idle")
	ch <- prometheus.MustNewConstMetric(xormPoolConnectionsDesc, prometheus.GaugeValue, float64(stats.MaxOpenConnections), name, role, "max_open")
	ch <- prometheus.MustNewConstMetric(xormPoolWaitCountDesc, prometheus.CounterValue, float64(stats.WaitCount), name, role)
	ch <- prometheus.MustNewConstMetric(xormPoolWaitDurationDesc, prometheus.CounterValue, stats.WaitDuration.Seconds(), name, role)
}

func beginXormPingObserve(name string) func(err error) {
	dbName := strings.TrimSpace(name)
	if dbName == "" {
		dbName = "default"
	}
	start := time.Now()
	return func(err error) {
		xormPingTotal.WithLabelValues(dbName, xormPingResult(err)).Inc()
		xormPingDuration.WithLabelValues(dbName).Observe(time.Since(start).Seconds())
		if err != nil {
			xormPingErrors.WithLabelValues(dbName).Inc()
			if xormPingIsTimeout(err) {
				xormPingTimeouts.WithLabelValues(dbName).Inc()
			}
		}
	}
}

func xormPingResult(err error) string {
	if err == nil {
		return "ok"
	}
	if xormPingIsTimeout(err) {
		return "timeout"
	}
	return "error"
}

func xormPingIsTimeout(err error) bool {
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
