package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/api/response"
)

type SystemHandler struct {
	now func() time.Time
}

func NewSystemHandler(now func() time.Time) *SystemHandler {
	return &SystemHandler{
		now: now,
	}
}

func (h *SystemHandler) Ping(c *gin.Context) {
	response.OK(c, gin.H{
		"pong":      "snowpanel",
		"timestamp": h.now().UTC().Format(time.RFC3339),
	})
}
