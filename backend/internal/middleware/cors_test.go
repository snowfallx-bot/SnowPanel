package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSHandlesPreflight(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS())
	router.OPTIONS("/api/v1/auth/login", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://203.0.113.10:5173")
	req.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://203.0.113.10:5173" {
		t.Fatalf("unexpected allow origin: %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Headers"); got != "authorization,content-type" {
		t.Fatalf("unexpected allow headers: %q", got)
	}
}

func TestCORSAllowsSimpleRequestWithoutOverridingStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS())
	router.POST("/api/v1/auth/login", func(c *gin.Context) {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid"})
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://198.51.100.25:5173")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://198.51.100.25:5173" {
		t.Fatalf("unexpected allow origin: %q", got)
	}
}
