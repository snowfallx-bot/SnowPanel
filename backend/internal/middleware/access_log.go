package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func AccessLog(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		requestID, _ := c.Get(RequestIDKey)
		spanContext := trace.SpanContextFromContext(c.Request.Context())
		log.Info(
			"http_request",
			zap.String("request_id", toString(requestID)),
			zap.String("trace_id", traceID(spanContext)),
			zap.String("span_id", spanID(spanContext)),
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", time.Since(start)),
			zap.String("user_agent", c.Request.UserAgent()),
		)
	}
}

func toString(value interface{}) string {
	v, ok := value.(string)
	if !ok {
		return ""
	}
	return v
}

func traceID(spanContext trace.SpanContext) string {
	if !spanContext.IsValid() {
		return ""
	}
	return spanContext.TraceID().String()
}

func spanID(spanContext trace.SpanContext) string {
	if !spanContext.IsValid() {
		return ""
	}
	return spanContext.SpanID().String()
}
