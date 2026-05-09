package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/requestctx"
)

func TestRequestIDStoresValueInRequestContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestID())
	router.GET("/ping", func(c *gin.Context) {
		requestID, ok := requestctx.RequestID(c.Request.Context())
		if !ok {
			t.Fatal("request context missing request id")
		}
		c.String(http.StatusOK, requestID)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-Request-ID", "req-ctx-001")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if recorder.Body.String() != "req-ctx-001" {
		t.Fatalf("expected request id in handler context, got %q", recorder.Body.String())
	}
}
