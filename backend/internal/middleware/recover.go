package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/api/response"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func Recover(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				span := trace.SpanFromContext(c.Request.Context())
				if span != nil {
					err := fmt.Errorf("panic: %v", rec)
					span.RecordError(err)
					span.SetStatus(otelcodes.Error, err.Error())
				}
				log.Error(
					"panic recovered",
					zap.Any("panic", rec),
					zap.ByteString("stack", debug.Stack()),
				)
				response.Fail(c, http.StatusInternalServerError, 1000, "internal server error")
				c.Abort()
			}
		}()

		c.Next()
	}
}
