package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	appmetrics "github.com/snowfallx-bot/SnowPanel/backend/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetricsCollectsRouteTemplateLabels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry := prometheus.NewRegistry()
	metrics := appmetrics.New(registry)

	router := gin.New()
	router.Use(Metrics(metrics))
	router.GET("/api/v1/services/:name/start", func(c *gin.Context) {
		c.Status(http.StatusAccepted)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/services/nginx/start", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", recorder.Code)
	}

	metricText := testutil.CollectAndCount(metrics.HTTPRequestsTotal)
	if metricText != 1 {
		t.Fatalf("expected 1 collected counter metric, got %d", metricText)
	}

	counter := testutil.ToFloat64(metrics.HTTPRequestsTotal.WithLabelValues(
		http.MethodGet,
		"/api/v1/services/:name/start",
		"202",
	))
	if counter != 1 {
		t.Fatalf("expected counter 1, got %f", counter)
	}

	histogramCount := testutil.CollectAndCount(metrics.HTTPRequestDuration)
	if histogramCount != 1 {
		t.Fatalf("expected 1 collected histogram metric, got %d", histogramCount)
	}
}

func TestMetricsUsesUnmatchedRouteLabel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry := prometheus.NewRegistry()
	metrics := appmetrics.New(registry)

	router := gin.New()
	router.Use(Metrics(metrics))

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}

	counter := testutil.ToFloat64(metrics.HTTPRequestsTotal.WithLabelValues(
		http.MethodGet,
		"unmatched",
		"404",
	))
	if counter != 1 {
		t.Fatalf("expected unmatched counter 1, got %f", counter)
	}
}

func TestMetricsDoesNotLeakRawPathInLabels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry := prometheus.NewRegistry()
	metrics := appmetrics.New(registry)

	router := gin.New()
	router.Use(Metrics(metrics))
	router.GET("/api/v1/tasks/:id", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/task-123", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	for _, family := range metricFamilies {
		if family.GetName() != "snowpanel_http_requests_total" {
			continue
		}
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == "route" && strings.Contains(label.GetValue(), "task-123") {
					t.Fatalf("raw path leaked into route label: %s", label.GetValue())
				}
			}
		}
	}
}
