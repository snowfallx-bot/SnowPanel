package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/api/response"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/service"
)

type HostHandler struct {
	hostService  service.HostService
	auditService service.AuditService
}

func NewHostHandler(hostService service.HostService, auditService service.AuditService) *HostHandler {
	return &HostHandler{hostService: hostService, auditService: auditService}
}

func (h *HostHandler) ListHosts(c *gin.Context) {
	result, err := h.hostService.ListHosts(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *HostHandler) GetHost(c *gin.Context) {
	id, ok := parseHostIDParam(c)
	if !ok {
		return
	}

	result, err := h.hostService.GetHost(c.Request.Context(), id)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *HostHandler) CreateHost(c *gin.Context) {
	var req dto.CreateHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request body")
		return
	}

	summary := hostAuditSummary(map[string]any{
		"op":      "create",
		"name":    req.Name,
		"address": req.Address,
		"port":    req.Port,
	})
	result, err := h.hostService.CreateHost(c.Request.Context(), req)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "hosts",
			Action:         "create",
			TargetType:     "host",
			TargetID:       req.Address,
			RequestSummary: summary,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "hosts",
		Action:         "create",
		TargetType:     "host",
		TargetID:       strconv.FormatInt(result.ID, 10),
		RequestSummary: summary,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "create success",
	})
	response.OK(c, result)
}

func (h *HostHandler) UpdateHost(c *gin.Context) {
	id, ok := parseHostIDParam(c)
	if !ok {
		return
	}

	var req dto.UpdateHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request body")
		return
	}

	summary := hostAuditSummary(map[string]any{
		"op":      "update",
		"id":      id,
		"name":    req.Name,
		"address": req.Address,
		"port":    req.Port,
	})
	result, err := h.hostService.UpdateHost(c.Request.Context(), id, req)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "hosts",
			Action:         "update",
			TargetType:     "host",
			TargetID:       strconv.FormatInt(id, 10),
			RequestSummary: summary,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "hosts",
		Action:         "update",
		TargetType:     "host",
		TargetID:       strconv.FormatInt(result.ID, 10),
		RequestSummary: summary,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "update success",
	})
	response.OK(c, result)
}

func (h *HostHandler) CheckHost(c *gin.Context) {
	id, ok := parseHostIDParam(c)
	if !ok {
		return
	}

	summary := hostAuditSummary(map[string]any{"op": "check", "id": id})
	result, err := h.hostService.CheckHost(c.Request.Context(), id)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "hosts",
			Action:         "check",
			TargetType:     "host",
			TargetID:       strconv.FormatInt(id, 10),
			RequestSummary: summary,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "hosts",
		Action:         "check",
		TargetType:     "host",
		TargetID:       strconv.FormatInt(result.ID, 10),
		RequestSummary: summary,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "check success",
	})
	response.OK(c, result)
}

func (h *HostHandler) EnableHost(c *gin.Context) {
	h.toggleHost(c, true)
}

func (h *HostHandler) DisableHost(c *gin.Context) {
	h.toggleHost(c, false)
}

func (h *HostHandler) toggleHost(c *gin.Context, enabled bool) {
	id, ok := parseHostIDParam(c)
	if !ok {
		return
	}

	action := "disable"
	if enabled {
		action = "enable"
	}
	summary := hostAuditSummary(map[string]any{"op": action, "id": id})

	var (
		result dto.Host
		err    error
	)
	if enabled {
		result, err = h.hostService.EnableHost(c.Request.Context(), id)
	} else {
		result, err = h.hostService.DisableHost(c.Request.Context(), id)
	}
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "hosts",
			Action:         action,
			TargetType:     "host",
			TargetID:       strconv.FormatInt(id, 10),
			RequestSummary: summary,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "hosts",
		Action:         action,
		TargetType:     "host",
		TargetID:       strconv.FormatInt(result.ID, 10),
		RequestSummary: summary,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  action + " success",
	})
	response.OK(c, result)
}

func parseHostIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid host id")
		return 0, false
	}
	return id, true
}

func hostAuditSummary(fields map[string]any) string {
	encoded, err := json.Marshal(fields)
	if err != nil {
		return `{"op":"host"}`
	}
	return string(encoded)
}
