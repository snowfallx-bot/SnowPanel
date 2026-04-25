package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func TraceRequestFields() gin.HandlerFunc {
	return func(c *gin.Context) {
		span := trace.SpanFromContext(c.Request.Context())
		if span != nil {
			if requestID, ok := c.Get(RequestIDKey); ok {
				span.SetAttributes(attribute.String("snowpanel.request_id", toString(requestID)))
			}
		}
		c.Next()
	}
}
