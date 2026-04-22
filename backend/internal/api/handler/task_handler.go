package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/api/response"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/middleware"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/service"
)

type TaskHandler struct {
	taskService  service.TaskService
	auditService service.AuditService
}

func NewTaskHandler(taskService service.TaskService, auditService service.AuditService) *TaskHandler {
	return &TaskHandler{
		taskService:  taskService,
		auditService: auditService,
	}
}

func (h *TaskHandler) CreateDockerRestartTask(c *gin.Context) {
	var req dto.CreateDockerRestartTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request body")
		return
	}

	var userIDPtr *int64
	if userID, ok := middleware.GetCurrentUserID(c); ok {
		userIDPtr = &userID
	}
	username, _ := middleware.GetCurrentUsername(c)

	result, err := h.taskService.CreateDockerRestartTask(c.Request.Context(), req, userIDPtr, username)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "tasks",
			Action:         "create_docker_restart",
			TargetType:     "task",
			TargetID:       req.ContainerID,
			RequestSummary: fmt.Sprintf(`{"operation":"docker.restart","container_id":"%s"}`, req.ContainerID),
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "tasks",
		Action:         "create_docker_restart",
		TargetType:     "task",
		TargetID:       strconv.FormatInt(result.ID, 10),
		RequestSummary: fmt.Sprintf(`{"operation":"docker.restart","container_id":"%s"}`, req.ContainerID),
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "task created",
	})
	response.OK(c, result)
}

func (h *TaskHandler) CreateServiceRestartTask(c *gin.Context) {
	var req dto.CreateServiceRestartTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid request body")
		return
	}

	var userIDPtr *int64
	if userID, ok := middleware.GetCurrentUserID(c); ok {
		userIDPtr = &userID
	}
	username, _ := middleware.GetCurrentUsername(c)

	result, err := h.taskService.CreateServiceRestartTask(c.Request.Context(), req, userIDPtr, username)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "tasks",
			Action:         "create_service_restart",
			TargetType:     "task",
			TargetID:       req.ServiceName,
			RequestSummary: fmt.Sprintf(`{"operation":"service.restart","service_name":"%s"}`, req.ServiceName),
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "tasks",
		Action:         "create_service_restart",
		TargetType:     "task",
		TargetID:       strconv.FormatInt(result.ID, 10),
		RequestSummary: fmt.Sprintf(`{"operation":"service.restart","service_name":"%s"}`, req.ServiceName),
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "task created",
	})
	response.OK(c, result)
}

func (h *TaskHandler) ListTasks(c *gin.Context) {
	var query dto.ListTasksQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid query params")
		return
	}

	result, err := h.taskService.ListTasks(c.Request.Context(), query)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *TaskHandler) GetTaskDetail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid task id")
		return
	}

	result, svcErr := h.taskService.GetTaskDetail(c.Request.Context(), id)
	if svcErr != nil {
		response.FromError(c, svcErr)
		return
	}
	response.OK(c, result)
}

func (h *TaskHandler) CancelTask(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid task id")
		return
	}

	username, _ := middleware.GetCurrentUsername(c)
	if err := h.taskService.CancelTask(c.Request.Context(), id, username); err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "tasks",
			Action:         "cancel",
			TargetType:     "task",
			TargetID:       strconv.FormatInt(id, 10),
			RequestSummary: `{"operation":"cancel"}`,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "tasks",
		Action:         "cancel",
		TargetType:     "task",
		TargetID:       strconv.FormatInt(id, 10),
		RequestSummary: `{"operation":"cancel"}`,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "task canceled",
	})
	response.OK(c, gin.H{"id": id, "status": service.TaskStatusCanceled})
}

func (h *TaskHandler) RetryTask(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Fail(c, http.StatusBadRequest, apperror.ErrBadRequest.Code, "invalid task id")
		return
	}

	var userIDPtr *int64
	if userID, ok := middleware.GetCurrentUserID(c); ok {
		userIDPtr = &userID
	}
	username, _ := middleware.GetCurrentUsername(c)

	result, svcErr := h.taskService.RetryTask(c.Request.Context(), id, userIDPtr, username)
	if svcErr != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "tasks",
			Action:         "retry",
			TargetType:     "task",
			TargetID:       strconv.FormatInt(id, 10),
			RequestSummary: `{"operation":"retry"}`,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  svcErr.Error(),
		})
		response.FromError(c, svcErr)
		return
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "tasks",
		Action:         "retry",
		TargetType:     "task",
		TargetID:       strconv.FormatInt(result.ID, 10),
		RequestSummary: `{"operation":"retry"}`,
		Success:        true,
		ResultCode:     "ok",
		ResultMessage:  "task retried",
	})
	response.OK(c, result)
}
