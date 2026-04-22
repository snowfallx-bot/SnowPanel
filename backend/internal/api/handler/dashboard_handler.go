package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/api/response"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/service"
)

type DashboardHandler struct {
	dashboardService service.DashboardService
}

func NewDashboardHandler(dashboardService service.DashboardService) *DashboardHandler {
	return &DashboardHandler{
		dashboardService: dashboardService,
	}
}

func (h *DashboardHandler) Summary(c *gin.Context) {
	summary, err := h.dashboardService.GetSummary(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, summary)
}
