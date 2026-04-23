package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/api/response"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db          *gorm.DB
	agentClient grpcclient.AgentClient
}

type dependencyStatus struct {
	Database string
	Agent    string
	Details  map[string]string
}

func NewHealthHandler(db *gorm.DB, agentClient grpcclient.AgentClient) *HealthHandler {
	return &HealthHandler{
		db:          db,
		agentClient: agentClient,
	}
}

func (h *HealthHandler) Health(c *gin.Context) {
	checks := h.collectDependencyStatus(c.Request.Context())
	status := "up"
	if checks.Database == "down" || checks.Agent == "down" {
		status = "degraded"
	}

	data := gin.H{
		"service": "backend",
		"status":  status,
		"checks": gin.H{
			"database": checks.Database,
			"agent":    checks.Agent,
		},
	}
	if len(checks.Details) > 0 {
		data["details"] = checks.Details
	}
	response.OK(c, data)
}

func (h *HealthHandler) Readiness(c *gin.Context) {
	checks := h.collectDependencyStatus(c.Request.Context())
	ready := checks.Database != "down" && checks.Agent != "down"

	data := gin.H{
		"service": "backend",
		"status":  "ready",
		"checks": gin.H{
			"database": checks.Database,
			"agent":    checks.Agent,
		},
	}
	if len(checks.Details) > 0 {
		data["details"] = checks.Details
	}

	if !ready {
		c.JSON(http.StatusServiceUnavailable, response.Envelope{
			Code:    1002,
			Message: "service not ready",
			Data:    data,
		})
		return
	}

	response.OK(c, data)
}

func (h *HealthHandler) collectDependencyStatus(ctx context.Context) dependencyStatus {
	result := dependencyStatus{
		Database: "skipped",
		Agent:    "skipped",
		Details:  map[string]string{},
	}

	if h.db != nil {
		sqlDB, err := h.db.DB()
		if err != nil {
			result.Database = "down"
			result.Details["database"] = err.Error()
		} else {
			pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()

			if err := sqlDB.PingContext(pingCtx); err != nil {
				result.Database = "down"
				result.Details["database"] = err.Error()
			} else {
				result.Database = "up"
			}
		}
	}

	if h.agentClient != nil {
		agentCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		status, err := h.agentClient.CheckHealth(agentCtx)
		if err != nil {
			result.Agent = "down"
			result.Details["agent"] = err.Error()
		} else if !strings.EqualFold(strings.TrimSpace(status), "SERVING") {
			result.Agent = "down"
			result.Details["agent"] = "unexpected health status: " + status
		} else {
			result.Agent = "up"
		}
	}

	if len(result.Details) == 0 {
		result.Details = nil
	}

	return result
}
