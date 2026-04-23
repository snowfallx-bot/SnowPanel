package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/api/response"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/service"
)

type CronHandler struct {
	cronService  service.CronService
	auditService service.AuditService
}

func NewCronHandler(cronService service.CronService, auditService service.AuditService) *CronHandler {
	return &CronHandler{
		cronService:  cronService,
		auditService: auditService,
	}
}

func (h *CronHandler) ListTasks(c *gin.Context) {
	summary := cronAuditSummary(map[string]any{
		"op": "list",
	})
	result, err := h.cronService.ListTasks(c.Request.Context())
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "cron",
			Action:         "list",
			TargetType:     "task",
			TargetID:       "*",
			RequestSummary: summary,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "cron",
		Action:         "list",
		TargetType:     "task",
		TargetID:       "*",
		RequestSummary: summary,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "list success",
	})
	response.OK(c, result)
}

func (h *CronHandler) CreateTask(c *gin.Context) {
	var req dto.CreateCronTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request payload")
		return
	}

	summary := cronAuditSummary(map[string]any{
		"op":         "create",
		"expression": req.Expression,
		"command":    req.Command,
		"enabled":    req.Enabled,
	})
	result, err := h.cronService.CreateTask(c.Request.Context(), req)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "cron",
			Action:         "create",
			TargetType:     "task",
			TargetID:       req.Command,
			RequestSummary: summary,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "cron",
		Action:         "create",
		TargetType:     "task",
		TargetID:       result.Task.ID,
		RequestSummary: summary,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "create success",
	})
	response.OK(c, result)
}

func (h *CronHandler) UpdateTask(c *gin.Context) {
	var path dto.UpdateCronTaskPath
	if err := c.ShouldBindUri(&path); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid task id")
		return
	}

	var req dto.UpdateCronTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request payload")
		return
	}

	summary := cronAuditSummary(map[string]any{
		"op":         "update",
		"id":         path.ID,
		"expression": req.Expression,
		"command":    req.Command,
		"enabled":    req.Enabled,
	})
	result, err := h.cronService.UpdateTask(c.Request.Context(), path.ID, req)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "cron",
			Action:         "update",
			TargetType:     "task",
			TargetID:       path.ID,
			RequestSummary: summary,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "cron",
		Action:         "update",
		TargetType:     "task",
		TargetID:       result.Task.ID,
		RequestSummary: summary,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "update success",
	})
	response.OK(c, result)
}

func (h *CronHandler) DeleteTask(c *gin.Context) {
	var path dto.DeleteCronTaskPath
	if err := c.ShouldBindUri(&path); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid task id")
		return
	}

	summary := cronAuditSummary(map[string]any{
		"op": "delete",
		"id": path.ID,
	})
	result, err := h.cronService.DeleteTask(c.Request.Context(), path.ID)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "cron",
			Action:         "delete",
			TargetType:     "task",
			TargetID:       path.ID,
			RequestSummary: summary,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "cron",
		Action:         "delete",
		TargetType:     "task",
		TargetID:       result.ID,
		RequestSummary: summary,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "delete success",
	})
	response.OK(c, result)
}

func (h *CronHandler) SetEnabled(c *gin.Context) {
	var path dto.ToggleCronTaskPath
	if err := c.ShouldBindUri(&path); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid task id")
		return
	}

	var req dto.ToggleCronTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request payload")
		return
	}

	summary := cronAuditSummary(map[string]any{
		"op":      "set_enabled",
		"id":      path.ID,
		"enabled": req.Enabled,
	})
	result, err := h.cronService.SetEnabled(c.Request.Context(), path.ID, req.Enabled)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "cron",
			Action:         "set_enabled",
			TargetType:     "task",
			TargetID:       path.ID,
			RequestSummary: summary,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "cron",
		Action:         "set_enabled",
		TargetType:     "task",
		TargetID:       result.Task.ID,
		RequestSummary: summary,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "set enabled success",
	})
	response.OK(c, result)
}

func (h *CronHandler) EnableTask(c *gin.Context) {
	h.toggle(c, true)
}

func (h *CronHandler) DisableTask(c *gin.Context) {
	h.toggle(c, false)
}

func (h *CronHandler) toggle(c *gin.Context, enabled bool) {
	var path dto.ToggleCronTaskPath
	if err := c.ShouldBindUri(&path); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid task id")
		return
	}

	action := "disable"
	if enabled {
		action = "enable"
	}
	summary := cronAuditSummary(map[string]any{
		"op":      action,
		"id":      path.ID,
		"enabled": enabled,
	})

	result, err := h.cronService.SetEnabled(c.Request.Context(), path.ID, enabled)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "cron",
			Action:         action,
			TargetType:     "task",
			TargetID:       path.ID,
			RequestSummary: summary,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "cron",
		Action:         action,
		TargetType:     "task",
		TargetID:       result.Task.ID,
		RequestSummary: summary,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  action + " success",
	})
	response.OK(c, result)
}

func cronAuditSummary(fields map[string]any) string {
	encoded, err := json.Marshal(fields)
	if err != nil {
		return `{"op":"cron"}`
	}
	return string(encoded)
}
