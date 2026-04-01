package redis

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var redisDurationBuckets = []float64{
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
	redisCommandsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "goone_redis_commands_total",
		Help: "Total Redis commands by instance, command, and result.",
	}, []string{"instance", "cmd", "result"})
	redisCommandDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "goone_redis_command_duration_seconds",
		Help:    "Latency distribution of Redis commands.",
		Buckets: redisDurationBuckets,
	}, []string{"instance", "cmd"})
	redisCommandErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "goone_redis_command_errors_total",
		Help: "Total Redis command errors.",
	}, []string{"instance", "cmd"})
	redisCommandTimeouts = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "goone_redis_command_timeouts_total",
		Help: "Total Redis command timeouts.",
	}, []string{"instance", "cmd"})
	redisCommandsInFlight = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "goone_redis_commands_in_flight",
		Help: "Current in-flight Redis commands.",
	}, []string{"instance", "cmd"})
)

func beginRedisObserve(instID uint32, cmd string) func(err error) {
	instanceLabel := fmt.Sprintf("%d", instID)
	cmdLabel := normalizeRedisCmd(cmd)
	redisCommandsInFlight.WithLabelValues(instanceLabel, cmdLabel).Inc()
	start := time.Now()
	return func(err error) {
		redisCommandsInFlight.WithLabelValues(instanceLabel, cmdLabel).Dec()
		redisCommandsTotal.WithLabelValues(instanceLabel, cmdLabel, redisResultLabel(err)).Inc()
		redisCommandDuration.WithLabelValues(instanceLabel, cmdLabel).Observe(time.Since(start).Seconds())
		if err != nil {
			redisCommandErrors.WithLabelValues(instanceLabel, cmdLabel).Inc()
			if redisIsTimeoutErr(err) {
				redisCommandTimeouts.WithLabelValues(instanceLabel, cmdLabel).Inc()
			}
		}
	}
}

func normalizeRedisCmd(cmd string) string {
	label := strings.TrimSpace(strings.ToUpper(cmd))
	if label == "" {
		return "UNKNOWN"
	}
	return label
}

func redisResultLabel(err error) string {
	if err == nil {
		return "ok"
	}
	if redisIsTimeoutErr(err) {
		return "timeout"
	}
	return "error"
}

func redisIsTimeoutErr(err error) bool {
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
