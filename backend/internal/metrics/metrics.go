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
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestsInFlight prometheus.Gauge
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
	}

	if registerer != nil {
		registerer.MustRegister(
			metrics.HTTPRequestsTotal,
			metrics.HTTPRequestDuration,
			metrics.HTTPRequestsInFlight,
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
