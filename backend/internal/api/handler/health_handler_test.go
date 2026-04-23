package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient"
)

type fakeHealthAgentClient struct {
	grpcclient.AgentClient
	status string
	err    error
}

func (f *fakeHealthAgentClient) CheckHealth(context.Context) (string, error) {
	return f.status, f.err
}

func TestHealthHandlerHealthReturnsUpWhenAgentIsServing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHealthHandler(nil, &fakeHealthAgentClient{status: "SERVING"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)

	handler.Health(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := decodeJSONBody(t, w.Body.Bytes())
	data := mustMap(t, body["data"])
	if data["status"] != "up" {
		t.Fatalf("expected status up, got %v", data["status"])
	}
	checks := mustMap(t, data["checks"])
	if checks["agent"] != "up" {
		t.Fatalf("expected agent check up, got %v", checks["agent"])
	}
}

func TestHealthHandlerHealthReturnsDegradedWhenAgentIsDown(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHealthHandler(nil, &fakeHealthAgentClient{err: errors.New("dial failed")})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)

	handler.Health(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := decodeJSONBody(t, w.Body.Bytes())
	data := mustMap(t, body["data"])
	if data["status"] != "degraded" {
		t.Fatalf("expected status degraded, got %v", data["status"])
	}
	checks := mustMap(t, data["checks"])
	if checks["agent"] != "down" {
		t.Fatalf("expected agent check down, got %v", checks["agent"])
	}
}

func TestHealthHandlerReadinessReturnsServiceUnavailableWhenAgentIsDown(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHealthHandler(nil, &fakeHealthAgentClient{err: errors.New("dial failed")})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ready", nil)

	handler.Readiness(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
	body := decodeJSONBody(t, w.Body.Bytes())
	if body["message"] != "service not ready" {
		t.Fatalf("expected readiness failure message, got %v", body["message"])
	}
}

func TestHealthHandlerReadinessReturnsOKWhenAgentIsServing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHealthHandler(nil, &fakeHealthAgentClient{status: "SERVING"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ready", nil)

	handler.Readiness(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := decodeJSONBody(t, w.Body.Bytes())
	data := mustMap(t, body["data"])
	if data["status"] != "ready" {
		t.Fatalf("expected status ready, got %v", data["status"])
	}
}

func decodeJSONBody(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	parsed := map[string]any{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}
	return parsed
}

func mustMap(t *testing.T, value any) map[string]any {
	t.Helper()
	parsed, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected object, got %T", value)
	}
	return parsed
}
