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
}

func NewDockerHandler(dockerService service.DockerService) *DockerHandler {
	return &DockerHandler{
		dockerService: dockerService,
	}
}

func (h *DockerHandler) ListContainers(c *gin.Context) {
	result, err := h.dockerService.ListContainers(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
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
		response.FromError(c, err)
		return
	}
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
		response.FromError(c, err)
		return
	}
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
		response.FromError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *DockerHandler) ListImages(c *gin.Context) {
	result, err := h.dockerService.ListImages(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, result)
}
