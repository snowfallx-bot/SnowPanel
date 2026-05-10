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
	TaskQueueDepth       prometheus.Gauge
	TasksRunning         prometheus.Gauge
	TasksCompletedTotal  *prometheus.CounterVec
	TaskDuration         *prometheus.HistogramVec
	TaskWorkerClaims     *prometheus.CounterVec
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
		TaskQueueDepth: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: Namespace,
				Subsystem: "tasks",
				Name:      "queue_depth",
				Help:      "Current number of pending durable tasks ready or waiting to run.",
			},
		),
		TasksRunning: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: Namespace,
				Subsystem: "tasks",
				Name:      "running",
				Help:      "Current number of durable tasks in running status.",
			},
		),
		TasksCompletedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: "tasks",
				Name:      "completed_total",
				Help:      "Total number of durable tasks completed by type and terminal status.",
			},
			[]string{"type", "status"},
		),
		TaskDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: Namespace,
				Subsystem: "tasks",
				Name:      "duration_seconds",
				Help:      "Duration of durable tasks by type and terminal status.",
				Buckets:   taskDurationBuckets(),
			},
			[]string{"type", "status"},
		),
		TaskWorkerClaims: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Subsystem: "task_worker",
				Name:      "claims_total",
				Help:      "Total number of durable task worker claim attempts by outcome.",
			},
			[]string{"outcome"},
		),
	}

	if registerer != nil {
		registerer.MustRegister(
			metrics.HTTPRequestsTotal,
			metrics.HTTPRequestDuration,
			metrics.HTTPRequestsInFlight,
			metrics.AgentRequestsTotal,
			metrics.AgentRequestDuration,
			metrics.TaskQueueDepth,
			metrics.TasksRunning,
			metrics.TasksCompletedTotal,
			metrics.TaskDuration,
			metrics.TaskWorkerClaims,
		)
	}

	metrics.initTaskMetricSeries()

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

func (s *Set) SetTaskQueueDepth(depth int64) {
	if s == nil {
		return
	}
	s.TaskQueueDepth.Set(float64(depth))
}

func (s *Set) SetTasksRunning(count int64) {
	if s == nil {
		return
	}
	s.TasksRunning.Set(float64(count))
}

func (s *Set) ObserveTaskCompleted(taskType, status string, duration time.Duration) {
	if s == nil {
		return
	}
	if taskType == "" {
		taskType = "unknown"
	}
	if status == "" {
		status = "unknown"
	}
	s.TasksCompletedTotal.WithLabelValues(taskType, status).Inc()
	s.TaskDuration.WithLabelValues(taskType, status).Observe(duration.Seconds())
}

func (s *Set) ObserveTaskWorkerClaim(outcome string) {
	if s == nil {
		return
	}
	if outcome == "" {
		outcome = "unknown"
	}
	s.TaskWorkerClaims.WithLabelValues(outcome).Inc()
}

func taskDurationBuckets() []float64 {
	return []float64{0.1, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300}
}

func (s *Set) initTaskMetricSeries() {
	for _, outcome := range []string{"success", "empty", "error"} {
		s.TaskWorkerClaims.WithLabelValues(outcome).Add(0)
	}
	for _, taskType := range []string{"docker_restart", "service_restart"} {
		for _, status := range []string{"success", "failed", "canceled"} {
			s.TasksCompletedTotal.WithLabelValues(taskType, status).Add(0)
			_ = s.TaskDuration.WithLabelValues(taskType, status)
		}
	}
}
