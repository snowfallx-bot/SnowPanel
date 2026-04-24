package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	appmetrics "github.com/snowfallx-bot/SnowPanel/backend/internal/metrics"
)

func Metrics(metrics *appmetrics.Set) gin.HandlerFunc {
	if metrics == nil {
		metrics = appmetrics.Default()
	}

	return func(c *gin.Context) {
		metrics.HTTPRequestsInFlight.Inc()
		startedAt := time.Now()
		defer metrics.HTTPRequestsInFlight.Dec()

		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}

		metrics.ObserveHTTPRequest(
			c.Request.Method,
			route,
			c.Writer.Status(),
			time.Since(startedAt),
		)
	}
}
