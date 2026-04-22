package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/api/response"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/service"
)

type ServiceHandler struct {
	serviceManager service.ServiceManagerService
	auditService   service.AuditService
}

func NewServiceHandler(
	serviceManager service.ServiceManagerService,
	auditService service.AuditService,
) *ServiceHandler {
	return &ServiceHandler{
		serviceManager: serviceManager,
		auditService:   auditService,
	}
}

func (h *ServiceHandler) ListServices(c *gin.Context) {
	var query dto.ListServicesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid query params")
		return
	}

	result, err := h.serviceManager.ListServices(c.Request.Context(), query)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "services",
			Action:         "list",
			TargetType:     "service",
			TargetID:       query.Keyword,
			RequestSummary: `{"op":"list_services"}`,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "services",
		Action:         "list",
		TargetType:     "service",
		TargetID:       query.Keyword,
		RequestSummary: `{"op":"list_services"}`,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "list success",
	})
	response.OK(c, result)
}

func (h *ServiceHandler) StartService(c *gin.Context) {
	var params dto.ServiceActionPath
	if err := c.ShouldBindUri(&params); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid service name")
		return
	}

	result, err := h.serviceManager.StartService(c.Request.Context(), params.Name)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "services",
			Action:         "start",
			TargetType:     "service",
			TargetID:       params.Name,
			RequestSummary: `{"op":"start_service"}`,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "services",
		Action:         "start",
		TargetType:     "service",
		TargetID:       params.Name,
		RequestSummary: `{"op":"start_service"}`,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "start success",
	})
	response.OK(c, result)
}

func (h *ServiceHandler) StopService(c *gin.Context) {
	var params dto.ServiceActionPath
	if err := c.ShouldBindUri(&params); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid service name")
		return
	}

	result, err := h.serviceManager.StopService(c.Request.Context(), params.Name)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "services",
			Action:         "stop",
			TargetType:     "service",
			TargetID:       params.Name,
			RequestSummary: `{"op":"stop_service"}`,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "services",
		Action:         "stop",
		TargetType:     "service",
		TargetID:       params.Name,
		RequestSummary: `{"op":"stop_service"}`,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "stop success",
	})
	response.OK(c, result)
}

func (h *ServiceHandler) RestartService(c *gin.Context) {
	var params dto.ServiceActionPath
	if err := c.ShouldBindUri(&params); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid service name")
		return
	}

	result, err := h.serviceManager.RestartService(c.Request.Context(), params.Name)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "services",
			Action:         "restart",
			TargetType:     "service",
			TargetID:       params.Name,
			RequestSummary: `{"op":"restart_service"}`,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "services",
		Action:         "restart",
		TargetType:     "service",
		TargetID:       params.Name,
		RequestSummary: `{"op":"restart_service"}`,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "restart success",
	})
	response.OK(c, result)
}
