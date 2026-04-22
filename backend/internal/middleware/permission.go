package middleware

import (
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
)

func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		username, _ := GetCurrentUsername(c)
		if username == "admin" {
			c.Next()
			return
		}

		permissions, ok := GetCurrentPermissions(c)
		if !ok || !slices.Contains(permissions, permission) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    apperror.ErrPermissionDenied.Code,
				"message": apperror.ErrPermissionDenied.Message,
				"data": gin.H{
					"required_permission": permission,
				},
			})
			return
		}

		c.Next()
	}
}
