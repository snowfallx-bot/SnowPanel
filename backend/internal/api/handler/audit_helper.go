package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/middleware"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/service"
)

func recordAudit(c *gin.Context, auditService service.AuditService, input dto.RecordAuditInput) {
	if auditService == nil {
		return
	}

	if input.UserID == nil {
		if userID, ok := middleware.GetCurrentUserID(c); ok {
			input.UserID = &userID
		}
	}
	if input.Username == "" {
		if username, ok := middleware.GetCurrentUsername(c); ok {
			input.Username = username
		}
	}
	if input.IP == "" {
		input.IP = c.ClientIP()
	}
	auditService.Record(c.Request.Context(), input)
}
