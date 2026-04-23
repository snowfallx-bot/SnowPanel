package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	defaultCORSAllowMethods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	defaultCORSAllowHeaders = "Authorization, Content-Type, X-Requested-With, X-Request-Id"
	defaultCORSExposeHeaders = "X-Request-Id"
	defaultCORSMaxAge = "600"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", defaultCORSAllowMethods)
			c.Header("Access-Control-Expose-Headers", defaultCORSExposeHeaders)
			c.Header("Access-Control-Max-Age", defaultCORSMaxAge)

			requestHeaders := strings.TrimSpace(c.GetHeader("Access-Control-Request-Headers"))
			if requestHeaders != "" {
				c.Header("Access-Control-Allow-Headers", requestHeaders)
				c.Header("Vary", "Origin, Access-Control-Request-Headers")
			} else {
				c.Header("Access-Control-Allow-Headers", defaultCORSAllowHeaders)
			}
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
