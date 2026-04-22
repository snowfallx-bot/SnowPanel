package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/api/response"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db *gorm.DB
}

func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Health(c *gin.Context) {
	status := "up"
	if h.db != nil {
		sqlDB, err := h.db.DB()
		if err != nil {
			response.Fail(c, http.StatusServiceUnavailable, 1001, "database unavailable")
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := sqlDB.PingContext(ctx); err != nil {
			status = "degraded"
		}
	}

	response.OK(c, gin.H{
		"service": "backend",
		"status":  status,
	})
}
