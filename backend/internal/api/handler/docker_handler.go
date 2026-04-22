package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/api/response"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/service"
)

type DockerHandler struct {
	dockerService service.DockerService
	auditService  service.AuditService
}

func NewDockerHandler(
	dockerService service.DockerService,
	auditService service.AuditService,
) *DockerHandler {
	return &DockerHandler{
		dockerService: dockerService,
		auditService:  auditService,
	}
}

func (h *DockerHandler) ListContainers(c *gin.Context) {
	result, err := h.dockerService.ListContainers(c.Request.Context())
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "docker",
			Action:         "list_containers",
			TargetType:     "docker_container",
			TargetID:       "",
			RequestSummary: `{"op":"list_containers"}`,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "docker",
		Action:         "list_containers",
		TargetType:     "docker_container",
		TargetID:       "",
		RequestSummary: `{"op":"list_containers"}`,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "list success",
	})
	response.OK(c, result)
}

func (h *DockerHandler) StartContainer(c *gin.Context) {
	var params dto.DockerContainerActionPath
	if err := c.ShouldBindUri(&params); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid container id")
		return
	}

	result, err := h.dockerService.StartContainer(c.Request.Context(), params.ID)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "docker",
			Action:         "start_container",
			TargetType:     "docker_container",
			TargetID:       params.ID,
			RequestSummary: `{"op":"start_container"}`,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "docker",
		Action:         "start_container",
		TargetType:     "docker_container",
		TargetID:       params.ID,
		RequestSummary: `{"op":"start_container"}`,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "start success",
	})
	response.OK(c, result)
}

func (h *DockerHandler) StopContainer(c *gin.Context) {
	var params dto.DockerContainerActionPath
	if err := c.ShouldBindUri(&params); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid container id")
		return
	}

	result, err := h.dockerService.StopContainer(c.Request.Context(), params.ID)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "docker",
			Action:         "stop_container",
			TargetType:     "docker_container",
			TargetID:       params.ID,
			RequestSummary: `{"op":"stop_container"}`,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "docker",
		Action:         "stop_container",
		TargetType:     "docker_container",
		TargetID:       params.ID,
		RequestSummary: `{"op":"stop_container"}`,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "stop success",
	})
	response.OK(c, result)
}

func (h *DockerHandler) RestartContainer(c *gin.Context) {
	var params dto.DockerContainerActionPath
	if err := c.ShouldBindUri(&params); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid container id")
		return
	}

	result, err := h.dockerService.RestartContainer(c.Request.Context(), params.ID)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "docker",
			Action:         "restart_container",
			TargetType:     "docker_container",
			TargetID:       params.ID,
			RequestSummary: `{"op":"restart_container"}`,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "docker",
		Action:         "restart_container",
		TargetType:     "docker_container",
		TargetID:       params.ID,
		RequestSummary: `{"op":"restart_container"}`,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "restart success",
	})
	response.OK(c, result)
}

func (h *DockerHandler) ListImages(c *gin.Context) {
	result, err := h.dockerService.ListImages(c.Request.Context())
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "docker",
			Action:         "list_images",
			TargetType:     "docker_image",
			TargetID:       "",
			RequestSummary: `{"op":"list_images"}`,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}
	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "docker",
		Action:         "list_images",
		TargetType:     "docker_image",
		TargetID:       "",
		RequestSummary: `{"op":"list_images"}`,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "list success",
	})
	response.OK(c, result)
}
