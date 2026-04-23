package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/middleware"
)

type fakeCronService struct {
	listResult       dto.ListCronTasksResult
	listErr          error
	createResult     dto.CreateCronTaskResult
	createErr        error
	createReq        dto.CreateCronTaskRequest
	updateResult     dto.UpdateCronTaskResult
	updateErr        error
	deleteResult     dto.DeleteCronTaskResult
	deleteErr        error
	setEnabledResult dto.ToggleCronTaskResult
	setEnabledErr    error
	setEnabledID     string
	setEnabledValue  bool
}

func (s *fakeCronService) ListTasks(context.Context) (dto.ListCronTasksResult, error) {
	return s.listResult, s.listErr
}

func (s *fakeCronService) CreateTask(
	_ context.Context,
	req dto.CreateCronTaskRequest,
) (dto.CreateCronTaskResult, error) {
	s.createReq = req
	return s.createResult, s.createErr
}

func (s *fakeCronService) UpdateTask(
	context.Context,
	string,
	dto.UpdateCronTaskRequest,
) (dto.UpdateCronTaskResult, error) {
	return s.updateResult, s.updateErr
}

func (s *fakeCronService) DeleteTask(context.Context, string) (dto.DeleteCronTaskResult, error) {
	return s.deleteResult, s.deleteErr
}

func (s *fakeCronService) SetEnabled(
	_ context.Context,
	id string,
	enabled bool,
) (dto.ToggleCronTaskResult, error) {
	s.setEnabledID = id
	s.setEnabledValue = enabled
	return s.setEnabledResult, s.setEnabledErr
}

type fakeAuditService struct {
	records []dto.RecordAuditInput
}

func (s *fakeAuditService) Record(_ context.Context, input dto.RecordAuditInput) {
	s.records = append(s.records, input)
}

func (s *fakeAuditService) List(
	context.Context,
	dto.ListAuditLogsQuery,
) (dto.ListAuditLogsResult, error) {
	return dto.ListAuditLogsResult{}, errors.New("not implemented")
}

func TestCronHandlerCreateTaskRecordsAuditOnSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cronSvc := &fakeCronService{
		createResult: dto.CreateCronTaskResult{
			Task: dto.CronTask{
				ID:         "task-1",
				Expression: "*/5 * * * *",
				Command:    "backup",
				Enabled:    true,
			},
		},
	}
	auditSvc := &fakeAuditService{}
	handler := NewCronHandler(cronSvc, auditSvc)

	payload := `{"expression":"*/5 * * * *","command":"backup","enabled":true}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/cron", bytes.NewBufferString(payload))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middleware.CurrentUserIDKey, int64(7))
	c.Set(middleware.CurrentUsernameKey, "operator")

	handler.CreateTask(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if len(auditSvc.records) != 1 {
		t.Fatalf("expected one audit record, got %d", len(auditSvc.records))
	}
	record := auditSvc.records[0]
	if record.Module != "cron" || record.Action != "create" {
		t.Fatalf("unexpected audit module/action: %s/%s", record.Module, record.Action)
	}
	if record.TargetID != "task-1" {
		t.Fatalf("expected target id task-1, got %s", record.TargetID)
	}
	if !record.Success {
		t.Fatalf("expected success audit record")
	}
	if record.UserID == nil || *record.UserID != 7 {
		t.Fatalf("expected user id from context")
	}
	if record.Username != "operator" {
		t.Fatalf("expected username from context, got %s", record.Username)
	}

	summary := decodeCronSummary(t, record.RequestSummary)
	if summary["op"] != "create" {
		t.Fatalf("expected op=create in summary")
	}
	if summary["command"] != "backup" {
		t.Fatalf("expected command in summary")
	}
}

func TestCronHandlerCreateTaskRecordsAuditOnFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cronSvc := &fakeCronService{
		createErr: apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			errors.New("blocked shell token"),
		),
	}
	auditSvc := &fakeAuditService{}
	handler := NewCronHandler(cronSvc, auditSvc)

	payload := `{"expression":"*/5 * * * *","command":"backup|sh","enabled":true}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/cron", bytes.NewBufferString(payload))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateTask(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if len(auditSvc.records) != 1 {
		t.Fatalf("expected one audit record, got %d", len(auditSvc.records))
	}
	record := auditSvc.records[0]
	if record.Success {
		t.Fatalf("expected failed audit record")
	}
	if record.TargetID != "backup|sh" {
		t.Fatalf("expected request command as target id, got %s", record.TargetID)
	}
	summary := decodeCronSummary(t, record.RequestSummary)
	if summary["op"] != "create" {
		t.Fatalf("expected op=create in summary")
	}
}

func TestCronHandlerDisableTaskRecordsAuditAndUsesFalseFlag(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cronSvc := &fakeCronService{
		setEnabledResult: dto.ToggleCronTaskResult{
			Task: dto.CronTask{
				ID:         "task-9",
				Expression: "0 * * * *",
				Command:    "cleanup",
				Enabled:    false,
			},
		},
	}
	auditSvc := &fakeAuditService{}
	handler := NewCronHandler(cronSvc, auditSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/cron/task-9/disable", nil)
	c.Params = gin.Params{{Key: "id", Value: "task-9"}}

	handler.DisableTask(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if cronSvc.setEnabledID != "task-9" {
		t.Fatalf("expected task id to be passed to service")
	}
	if cronSvc.setEnabledValue {
		t.Fatalf("expected disable to pass enabled=false")
	}
	if len(auditSvc.records) != 1 {
		t.Fatalf("expected one audit record, got %d", len(auditSvc.records))
	}
	record := auditSvc.records[0]
	if record.Action != "disable" {
		t.Fatalf("expected disable action, got %s", record.Action)
	}
	if !record.Success {
		t.Fatalf("expected success audit record")
	}
	summary := decodeCronSummary(t, record.RequestSummary)
	if summary["op"] != "disable" {
		t.Fatalf("expected op=disable in summary")
	}
	if summary["enabled"] != false {
		t.Fatalf("expected enabled=false in summary")
	}
}

func decodeCronSummary(t *testing.T, raw string) map[string]any {
	t.Helper()
	out := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("failed to decode request summary: %v", err)
	}
	return out
}
