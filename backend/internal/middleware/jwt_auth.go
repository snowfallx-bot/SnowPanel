package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/service"
)

const (
	CurrentUserIDKey   = "current_user_id"
	CurrentUsernameKey = "current_username"
	CurrentRolesKey    = "current_roles"
	CurrentPermsKey    = "current_permissions"
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

		c.Set(CurrentUserIDKey, claims.UserID)
		c.Set(CurrentUsernameKey, claims.Username)
		c.Set(CurrentRolesKey, claims.Roles)
		c.Set(CurrentPermsKey, claims.Permissions)
		c.Next()
	}
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
