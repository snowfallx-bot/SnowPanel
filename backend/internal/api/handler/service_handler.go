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
}

func NewServiceHandler(serviceManager service.ServiceManagerService) *ServiceHandler {
	return &ServiceHandler{
		serviceManager: serviceManager,
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
		response.FromError(c, err)
		return
	}
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
		response.FromError(c, err)
		return
	}
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
		response.FromError(c, err)
		return
	}
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
		response.FromError(c, err)
		return
	}
	response.OK(c, result)
}
