package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/api/response"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/service"
)

type CronHandler struct {
	cronService service.CronService
}

func NewCronHandler(cronService service.CronService) *CronHandler {
	return &CronHandler{
		cronService: cronService,
	}
}

func (h *CronHandler) ListTasks(c *gin.Context) {
	result, err := h.cronService.ListTasks(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *CronHandler) CreateTask(c *gin.Context) {
	var req dto.CreateCronTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request payload")
		return
	}

	result, err := h.cronService.CreateTask(c.Request.Context(), req)
	if err != nil {
		response.FromError(c, err)
		return
	}
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

	result, err := h.cronService.UpdateTask(c.Request.Context(), path.ID, req)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *CronHandler) DeleteTask(c *gin.Context) {
	var path dto.DeleteCronTaskPath
	if err := c.ShouldBindUri(&path); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid task id")
		return
	}

	result, err := h.cronService.DeleteTask(c.Request.Context(), path.ID)
	if err != nil {
		response.FromError(c, err)
		return
	}
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

	result, err := h.cronService.SetEnabled(c.Request.Context(), path.ID, req.Enabled)
	if err != nil {
		response.FromError(c, err)
		return
	}
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

	result, err := h.cronService.SetEnabled(c.Request.Context(), path.ID, enabled)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, result)
}
