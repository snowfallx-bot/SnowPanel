package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/requestctx"
)

const RequestIDKey = "request_id"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Set(RequestIDKey, requestID)
		c.Request = c.Request.WithContext(requestctx.WithRequestID(c.Request.Context(), requestID))
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Next()
	}
}
