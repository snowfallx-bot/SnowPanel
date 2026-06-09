package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/hostctx"
)

const SelectedHostIDQueryKey = "host_id"

func OptionalHostSelection() gin.HandlerFunc {
	return func(c *gin.Context) {
		rawHostID := strings.TrimSpace(c.Query(SelectedHostIDQueryKey))
		if rawHostID == "" {
			c.Next()
			return
		}

		hostID, err := strconv.ParseInt(rawHostID, 10, 64)
		if err != nil || hostID <= 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"code":    apperror.ErrBadRequest.Code,
				"message": "invalid host_id",
				"data":    gin.H{},
			})
			return
		}

		c.Request = c.Request.WithContext(hostctx.WithHostID(c.Request.Context(), hostID))
		c.Next()
	}
}
