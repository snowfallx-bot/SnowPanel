package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/middleware"
)

type taskHandlerServiceStub struct {
	listQuery    dto.ListTasksQuery
	listResult   dto.ListTasksResult
	listErr      error
	cancelID     int64
	cancelUser   string
	cancelErr    error
	retryID      int64
	retryUserID  *int64
	retryUser    string
	retryResult  dto.CreateTaskResult
	retryErr     error
	detailResult dto.TaskDetail
	detailErr    error
}

func (s *taskHandlerServiceStub) CreateDockerRestartTask(
	context.Context,
	dto.CreateDockerRestartTaskRequest,
	*int64,
	string,
) (dto.CreateTaskResult, error) {
	return dto.CreateTaskResult{}, errors.New("not implemented")
}

func (s *taskHandlerServiceStub) CreateServiceRestartTask(
	context.Context,
	dto.CreateServiceRestartTaskRequest,
	*int64,
	string,
) (dto.CreateTaskResult, error) {
	return dto.CreateTaskResult{}, errors.New("not implemented")
}

func (s *taskHandlerServiceStub) CancelTask(_ context.Context, id int64, username string) error {
	s.cancelID = id
	s.cancelUser = username
	return s.cancelErr
}

func (s *taskHandlerServiceStub) RetryTask(
	_ context.Context,
	id int64,
	triggeredBy *int64,
	username string,
) (dto.CreateTaskResult, error) {
	s.retryID = id
	s.retryUserID = triggeredBy
	s.retryUser = username
	return s.retryResult, s.retryErr
}

func (s *taskHandlerServiceStub) ListTasks(
	_ context.Context,
	query dto.ListTasksQuery,
) (dto.ListTasksResult, error) {
	s.listQuery = query
	return s.listResult, s.listErr
}

func (s *taskHandlerServiceStub) GetTaskDetail(context.Context, int64) (dto.TaskDetail, error) {
	return s.detailResult, s.detailErr
}

type taskAuditRecorder struct {
	records []dto.RecordAuditInput
}

func (s *taskAuditRecorder) Record(_ context.Context, input dto.RecordAuditInput) {
	s.records = append(s.records, input)
}

func (s *taskAuditRecorder) List(
	context.Context,
	dto.ListAuditLogsQuery,
) (dto.ListAuditLogsResult, error) {
	return dto.ListAuditLogsResult{}, errors.New("not implemented")
}

func TestTaskHandlerListTasksPassesFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	taskSvc := &taskHandlerServiceStub{
		listResult: dto.ListTasksResult{
			Page: 2,
			Size: 10,
			Items: []dto.TaskSummary{
				{
					ID:       11,
					Type:     "service_restart",
					Status:   "running",
					Progress: 35,
				},
			},
		},
	}
	handler := NewTaskHandler(taskSvc, &taskAuditRecorder{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tasks?page=2&size=10&status=running&type=service_restart",
		nil,
	)

	handler.ListTasks(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if taskSvc.listQuery.Page != 2 || taskSvc.listQuery.Size != 10 {
		t.Fatalf("unexpected pagination query: %+v", taskSvc.listQuery)
	}
	if taskSvc.listQuery.Status != "running" || taskSvc.listQuery.Type != "service_restart" {
		t.Fatalf("unexpected filter query: %+v", taskSvc.listQuery)
	}
}

func TestTaskHandlerCancelTaskRecordsAuditOnSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	taskSvc := &taskHandlerServiceStub{}
	auditSvc := &taskAuditRecorder{}
	handler := NewTaskHandler(taskSvc, auditSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/tasks/12/cancel", nil)
	c.Params = gin.Params{{Key: "id", Value: "12"}}
	c.Set(middleware.CurrentUsernameKey, "operator")

	handler.CancelTask(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if taskSvc.cancelID != 12 || taskSvc.cancelUser != "operator" {
		t.Fatalf("unexpected cancel args id=%d user=%s", taskSvc.cancelID, taskSvc.cancelUser)
	}
	if len(auditSvc.records) != 1 {
		t.Fatalf("expected one audit record, got %d", len(auditSvc.records))
	}
	record := auditSvc.records[0]
	if record.Module != "tasks" || record.Action != "cancel" || !record.Success {
		t.Fatalf("unexpected audit record: %+v", record)
	}
	if record.TargetID != "12" {
		t.Fatalf("unexpected target id: %s", record.TargetID)
	}

	body := map[string]any{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected response data object")
	}
	status, _ := data["status"].(string)
	if status != "canceled" {
		t.Fatalf("expected canceled status, got %s", status)
	}
}

func TestTaskHandlerRetryTaskRecordsAuditOnFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	taskSvc := &taskHandlerServiceStub{
		retryErr: apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			errors.New("only failed tasks can be retried"),
		),
	}
	auditSvc := &taskAuditRecorder{}
	handler := NewTaskHandler(taskSvc, auditSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/tasks/3/retry", nil)
	c.Params = gin.Params{{Key: "id", Value: "3"}}
	userID := int64(7)
	c.Set(middleware.CurrentUserIDKey, userID)
	c.Set(middleware.CurrentUsernameKey, "operator")

	handler.RetryTask(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if taskSvc.retryID != 3 || taskSvc.retryUser != "operator" {
		t.Fatalf("unexpected retry args id=%d user=%s", taskSvc.retryID, taskSvc.retryUser)
	}
	if taskSvc.retryUserID == nil || *taskSvc.retryUserID != userID {
		t.Fatalf("expected retry user id to be propagated")
	}
	if len(auditSvc.records) != 1 {
		t.Fatalf("expected one audit record, got %d", len(auditSvc.records))
	}
	record := auditSvc.records[0]
	if record.Success {
		t.Fatalf("expected failed audit record")
	}
	if record.Action != "retry" || record.TargetID != strconv.FormatInt(taskSvc.retryID, 10) {
		t.Fatalf("unexpected audit action/target: %+v", record)
	}
}
