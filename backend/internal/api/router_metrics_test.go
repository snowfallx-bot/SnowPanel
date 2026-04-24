package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRouterExposesMetricsEndpoint(t *testing.T) {
	router := NewRouter(RouterDeps{})

	healthReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthRecorder := httptest.NewRecorder()
	router.ServeHTTP(healthRecorder, healthReq)
	if healthRecorder.Code != http.StatusOK {
		t.Fatalf("expected /health 200, got %d", healthRecorder.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "snowpanel_http_requests_total") {
		t.Fatalf("expected metrics body to contain requests counter")
	}
	if !strings.Contains(body, "snowpanel_http_request_duration_seconds") {
		t.Fatalf("expected metrics body to contain request duration histogram")
	}
	if !strings.Contains(body, "snowpanel_http_requests_in_flight") {
		t.Fatalf("expected metrics body to contain in-flight gauge")
	}
	if !strings.Contains(body, "route=\"/health\"") {
		t.Fatalf("expected metrics body to include /health route label")
	}
}
