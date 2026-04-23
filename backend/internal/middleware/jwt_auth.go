package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/service"
)

const (
	CurrentUserIDKey             = "current_user_id"
	CurrentUsernameKey           = "current_username"
	CurrentRolesKey              = "current_roles"
	CurrentPermsKey              = "current_permissions"
	CurrentMustChangePasswordKey = "current_must_change_password"
)

func JWTAuth(authService service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    apperror.ErrUnauthorized.Code,
				"message": apperror.ErrUnauthorized.Message,
				"data":    gin.H{},
			})
			return
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(authHeader, prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    apperror.ErrUnauthorized.Code,
				"message": apperror.ErrUnauthorized.Message,
				"data":    gin.H{},
			})
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
		claims, err := authService.ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    apperror.ErrTokenParse.Code,
				"message": apperror.ErrTokenParse.Message,
				"data":    gin.H{},
			})
			return
		}
		if err := authService.ValidateSession(c.Request.Context(), claims); err != nil {
			appErr, ok := apperror.As(err)
			if !ok {
				appErr = apperror.ErrTokenParse
			}
			c.AbortWithStatusJSON(appErr.HTTPStatus, gin.H{
				"code":    appErr.Code,
				"message": appErr.Message,
				"data":    gin.H{},
			})
			return
		}

		c.Set(CurrentUserIDKey, claims.UserID)
		c.Set(CurrentUsernameKey, claims.Username)
		c.Set(CurrentRolesKey, claims.Roles)
		c.Set(CurrentPermsKey, claims.Permissions)
		c.Set(CurrentMustChangePasswordKey, claims.MustChangePassword)
		if claims.MustChangePassword && !isPasswordChangeAllowedPath(c.Request.URL.Path) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    apperror.ErrPasswordChangeNeed.Code,
				"message": apperror.ErrPasswordChangeNeed.Message,
				"data":    gin.H{},
			})
			return
		}
		c.Next()
	}
}

func isPasswordChangeAllowedPath(path string) bool {
	return path == "/api/v1/auth/me" ||
		path == "/api/v1/auth/change-password" ||
		path == "/api/v1/auth/logout"
}

func GetCurrentUserID(c *gin.Context) (int64, bool) {
	value, exists := c.Get(CurrentUserIDKey)
	if !exists {
		return 0, false
	}
	userID, ok := value.(int64)
	return userID, ok
}

func GetCurrentUsername(c *gin.Context) (string, bool) {
	value, exists := c.Get(CurrentUsernameKey)
	if !exists {
		return "", false
	}
	username, ok := value.(string)
	return username, ok
}

func GetCurrentPermissions(c *gin.Context) ([]string, bool) {
	value, exists := c.Get(CurrentPermsKey)
	if !exists {
		return nil, false
	}
	permissions, ok := value.([]string)
	return permissions, ok
}

func GetCurrentMustChangePassword(c *gin.Context) (bool, bool) {
	value, exists := c.Get(CurrentMustChangePasswordKey)
	if !exists {
		return false, false
	}
	flag, ok := value.(bool)
	return flag, ok
}
