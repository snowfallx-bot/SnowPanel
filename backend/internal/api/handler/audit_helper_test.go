package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/hostctx"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/middleware"
)

type auditRecorderStub struct {
	records []dto.RecordAuditInput
}

func (s *auditRecorderStub) Record(_ context.Context, input dto.RecordAuditInput) {
	s.records = append(s.records, input)
}

func (s *auditRecorderStub) List(
	context.Context,
	dto.ListAuditLogsQuery,
) (dto.ListAuditLogsResult, error) {
	return dto.ListAuditLogsResult{}, errors.New("not implemented")
}

func TestRecordAuditInjectsSelectedHostContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	auditSvc := &auditRecorderStub{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/9/cancel?host_id=12", nil)
	c.Request = req.WithContext(hostctx.WithHostID(req.Context(), 12))
	c.Set(middleware.CurrentUserIDKey, int64(7))
	c.Set(middleware.CurrentUsernameKey, "operator")

	recordAudit(c, auditSvc, dto.RecordAuditInput{
		Module:     "tasks",
		Action:     "cancel",
		TargetType: "task",
		TargetID:   "9",
		Success:    true,
	})

	if len(auditSvc.records) != 1 {
		t.Fatalf("expected one audit record, got %d", len(auditSvc.records))
	}
	record := auditSvc.records[0]
	if record.HostID == nil || *record.HostID != 12 {
		t.Fatalf("expected host id 12, got %+v", record.HostID)
	}
	if record.UserID == nil || *record.UserID != 7 {
		t.Fatalf("expected user id 7, got %+v", record.UserID)
	}
	if record.Username != "operator" {
		t.Fatalf("expected username operator, got %q", record.Username)
	}
}
