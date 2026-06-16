package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/api/response"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/service"
)

type SettingsHandler struct {
	settingsService service.SettingsService
	auditService    service.AuditService
}

func NewSettingsHandler(settingsService service.SettingsService, auditService service.AuditService) *SettingsHandler {
	return &SettingsHandler{
		settingsService: settingsService,
		auditService:    auditService,
	}
}

func (h *SettingsHandler) ListSettings(c *gin.Context) {
	result, err := h.settingsService.ListSettings(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, dto.ListSettingsResponse{Items: result})
}

func (h *SettingsHandler) GetSetting(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "setting key is required")
		return
	}

	result, err := h.settingsService.GetSetting(c.Request.Context(), key)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "settings",
			Action:         "get",
			TargetType:     "setting",
			TargetID:       key,
			RequestSummary: `{"op":"get_setting","key":"` + key + `"}`,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "settings",
		Action:         "get",
		TargetType:     "setting",
		TargetID:       key,
		RequestSummary: `{"op":"get_setting","key":"` + key + `"}`,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "get success",
	})
	response.OK(c, result)
}

func (h *SettingsHandler) CreateSetting(c *gin.Context) {
	var req dto.CreateSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request body")
		return
	}

	summary := `{"op":"create_setting","key":"` + req.Key + `"}`
	result, err := h.settingsService.CreateSetting(c.Request.Context(), req)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "settings",
			Action:         "create",
			TargetType:     "setting",
			TargetID:       req.Key,
			RequestSummary: summary,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "settings",
		Action:         "create",
		TargetType:     "setting",
		TargetID:       req.Key,
		RequestSummary: summary,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "create success",
	})
	response.OK(c, result)
}

func (h *SettingsHandler) UpdateSetting(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "setting key is required")
		return
	}

	var req dto.UpdateSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request body")
		return
	}

	summary := `{"op":"update_setting","key":"` + key + `"}`
	result, err := h.settingsService.UpdateSetting(c.Request.Context(), key, req)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "settings",
			Action:         "update",
			TargetType:     "setting",
			TargetID:       key,
			RequestSummary: summary,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "settings",
		Action:         "update",
		TargetType:     "setting",
		TargetID:       key,
		RequestSummary: summary,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "update success",
	})
	response.OK(c, result)
}

func (h *SettingsHandler) DeleteSetting(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "setting key is required")
		return
	}

	summary := `{"op":"delete_setting","key":"` + key + `"}`
	if err := h.settingsService.DeleteSetting(c.Request.Context(), key); err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "settings",
			Action:         "delete",
			TargetType:     "setting",
			TargetID:       key,
			RequestSummary: summary,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "settings",
		Action:         "delete",
		TargetType:     "setting",
		TargetID:       key,
		RequestSummary: summary,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "delete success",
	})
	response.OK(c, gin.H{})
}

func parseSettingKeyParam(c *gin.Context) (string, bool) {
	key := c.Param("key")
	if key == "" {
		return "", false
	}
	return key, true
}
