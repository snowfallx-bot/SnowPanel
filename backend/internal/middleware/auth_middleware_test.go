package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/service"
)

type authServiceStub struct {
	claims service.TokenClaims
	err    error
}

func (s authServiceStub) EnsureDefaultAdmin(context.Context) error {
	return nil
}

func (s authServiceStub) Login(context.Context, dto.LoginRequest) (dto.LoginResponse, error) {
	return dto.LoginResponse{}, nil
}

func (s authServiceStub) Me(context.Context, int64) (dto.UserProfile, error) {
	return dto.UserProfile{}, nil
}

func (s authServiceStub) ChangePassword(
	context.Context,
	int64,
	dto.ChangePasswordRequest,
) (dto.LoginResponse, error) {
	return dto.LoginResponse{}, nil
}

func (s authServiceStub) ParseToken(string) (service.TokenClaims, error) {
	if s.err != nil {
		return service.TokenClaims{}, s.err
	}
	return s.claims, nil
}

func TestJWTAuthRejectsProtectedRouteWhenPasswordChangeRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/files/list", JWTAuth(authServiceStub{
		claims: service.TokenClaims{
			UserID:             100,
			Username:           "admin",
			Roles:              []string{"super_admin"},
			Permissions:        []string{"files.read"},
			MustChangePassword: true,
		},
	}), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/files/list", nil)
	req.Header.Set("Authorization", "Bearer fake-token")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if int(body["code"].(float64)) != apperror.ErrPasswordChangeNeed.Code {
		t.Fatalf("expected password-change-required code, got %v", body["code"])
	}
}

func TestJWTAuthAllowsChangePasswordRouteWhenPasswordChangeRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/auth/change-password", JWTAuth(authServiceStub{
		claims: service.TokenClaims{
			UserID:             100,
			Username:           "admin",
			Roles:              []string{"super_admin"},
			Permissions:        []string{"dashboard.read"},
			MustChangePassword: true,
		},
	}), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", nil)
	req.Header.Set("Authorization", "Bearer fake-token")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestJWTAuthRejectsMissingAuthorizationHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/protected", JWTAuth(authServiceStub{}), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if int(body["code"].(float64)) != apperror.ErrUnauthorized.Code {
		t.Fatalf("expected unauthorized code, got %v", body["code"])
	}
}

func TestJWTAuthInjectsClaimsOnSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/protected", JWTAuth(authServiceStub{
		claims: service.TokenClaims{
			UserID:      100,
			Username:    "tester",
			Roles:       []string{"operator"},
			Permissions: []string{"dashboard.read"},
		},
	}), func(c *gin.Context) {
		userID, _ := GetCurrentUserID(c)
		username, _ := GetCurrentUsername(c)
		c.JSON(http.StatusOK, gin.H{
			"user_id":  userID,
			"username": username,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer fake-token")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if int64(body["user_id"].(float64)) != 100 {
		t.Fatalf("unexpected user_id: %v", body["user_id"])
	}
	if body["username"].(string) != "tester" {
		t.Fatalf("unexpected username: %v", body["username"])
	}
}

func TestRequirePermissionRejectsUnauthorizedPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET(
		"/need-perm",
		func(c *gin.Context) {
			c.Set(CurrentUsernameKey, "operator")
			c.Set(CurrentPermsKey, []string{"files.read"})
			c.Next()
		},
		RequirePermission("docker.manage"),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/need-perm", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestRequirePermissionAllowsWhenPermissionPresent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET(
		"/need-perm",
		func(c *gin.Context) {
			c.Set(CurrentUsernameKey, "operator")
			c.Set(CurrentPermsKey, []string{"docker.manage"})
			c.Next()
		},
		RequirePermission("docker.manage"),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/need-perm", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}
