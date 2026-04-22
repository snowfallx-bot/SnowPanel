package handler

import (
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

func (h *TaskHandler) CreateDemoTask(c *gin.Context) {
	var userIDPtr *int64
	if userID, ok := middleware.GetCurrentUserID(c); ok {
		userIDPtr = &userID
	}
	username, _ := middleware.GetCurrentUsername(c)

	result, err := h.taskService.CreateDemoTask(c.Request.Context(), userIDPtr, username)
	if err != nil {
		recordAudit(c, h.auditService, dto.RecordAuditInput{
			Module:         "tasks",
			Action:         "create_demo",
			TargetType:     "task",
			TargetID:       "",
			RequestSummary: `{"type":"mock_backup"}`,
			Success:        false,
			ResultCode:     "failed",
			ResultMessage:  err.Error(),
		})
		response.FromError(c, err)
		return
	}

	recordAudit(c, h.auditService, dto.RecordAuditInput{
		Module:         "tasks",
		Action:         "create_demo",
		TargetType:     "task",
		TargetID:       strconv.FormatInt(result.ID, 10),
		RequestSummary: `{"type":"mock_backup"}`,
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
