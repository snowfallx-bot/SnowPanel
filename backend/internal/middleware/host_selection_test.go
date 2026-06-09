package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/hostctx"
)

func TestOptionalHostSelectionWritesHostIDIntoContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(OptionalHostSelection())
	router.GET("/test", func(c *gin.Context) {
		hostID, ok := hostctx.HostID(c.Request.Context())
		if !ok || hostID != 42 {
			t.Fatalf("expected host id 42 in context, got %d, %t", hostID, ok)
		}
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/test?host_id=42", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestOptionalHostSelectionRejectsInvalidHostID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(OptionalHostSelection())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/test?host_id=abc", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
