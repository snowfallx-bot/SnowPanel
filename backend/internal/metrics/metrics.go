package metrics

import (
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const Namespace = "snowpanel"

var (
	once sync.Once
	set  *Set
)

type Set struct {
	HTTPRequestsTotal    *prometheus.CounterVec
	HTTPRequestDuration  *prometheus.HistogramVec
	HTTPRequestsInFlight prometheus.Gauge
	AgentRequestsTotal   *prometheus.CounterVec
	AgentRequestDuration *prometheus.HistogramVec
}

func Default() *Set {
	once.Do(func() {
		set = New(prometheus.DefaultRegisterer)
	})
	return set
}

func New(registerer prometheus.Registerer) *Set {
	metrics := &Set{
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests processed by the backend.",
			},
			[]string{"method", "route", "status"},
		),
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: Namespace,
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "Latency of HTTP requests processed by the backend.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "route"},
		),
		HTTPRequestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: Namespace,
				Subsystem: "http",
				Name:      "requests_in_flight",
				Help:      "Current number of in-flight HTTP requests.",
			},
		),
		AgentRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: "agent",
				Name:      "requests_total",
				Help:      "Total number of core-agent RPC requests attempted by the backend.",
			},
			[]string{"rpc", "outcome", "transport"},
		),
		AgentRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: Namespace,
				Subsystem: "agent",
				Name:      "request_duration_seconds",
				Help:      "Latency of core-agent RPC requests attempted by the backend.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"rpc", "outcome", "transport"},
		),
	}

	if registerer != nil {
		registerer.MustRegister(
			metrics.HTTPRequestsTotal,
			metrics.HTTPRequestDuration,
			metrics.HTTPRequestsInFlight,
			metrics.AgentRequestsTotal,
			metrics.AgentRequestDuration,
		)
	}

	return metrics
}

func (s *Set) ObserveHTTPRequest(method, route string, statusCode int, duration time.Duration) {
	if s == nil {
		return
	}
	status := strconv.Itoa(statusCode)
	s.HTTPRequestsTotal.WithLabelValues(method, route, status).Inc()
	s.HTTPRequestDuration.WithLabelValues(method, route).Observe(duration.Seconds())
}

func (s *Set) ObserveAgentRequest(rpc string, transport bool, err error, duration time.Duration) {
	if s == nil {
		return
	}
	if rpc == "" {
		rpc = "unknown"
	}

	outcome := "success"
	if err != nil {
		outcome = "error"
	}
	transportLabel := "false"
	if transport {
		transportLabel = "true"
	}

	s.AgentRequestsTotal.WithLabelValues(rpc, outcome, transportLabel).Inc()
	s.AgentRequestDuration.WithLabelValues(rpc, outcome, transportLabel).Observe(duration.Seconds())
}
