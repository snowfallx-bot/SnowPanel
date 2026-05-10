package service

import (
	"context"
	"strings"
	"testing"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/repository"
)

type fakeAuditRepo struct {
	items []model.AuditLog
}

func (r *fakeAuditRepo) Create(_ context.Context, item *model.AuditLog) error {
	cloned := *item
	r.items = append(r.items, cloned)
	return nil
}

func (r *fakeAuditRepo) List(
	context.Context,
	repository.AuditListFilter,
) ([]model.AuditLog, int64, error) {
	return r.items, int64(len(r.items)), nil
}

func TestAuditRecordRedactsSensitiveFields(t *testing.T) {
	repo := &fakeAuditRepo{}
	service := NewAuditService(repo)

	service.Record(context.Background(), dto.RecordAuditInput{
		Username:       "admin",
		Module:         "settings",
		Action:         "update",
		TargetType:     "system_setting",
		TargetID:       "agent",
		RequestSummary: `{"token":"agent-token","nested":{"password":"plain"},"service_name":"nginx"}`,
		Success:        false,
		ResultCode:     "failed",
		ResultMessage:  "validation failed token=agent-token password=plain",
	})

	if len(repo.items) != 1 {
		t.Fatalf("expected one audit item, got %d", len(repo.items))
	}
	item := repo.items[0]
	if strings.Contains(item.RequestSummary, "agent-token") ||
		strings.Contains(item.RequestSummary, "plain") ||
		strings.Contains(item.ResultMessage, "agent-token") ||
		strings.Contains(item.ResultMessage, "plain") {
		t.Fatalf("audit item leaked sensitive data: %+v", item)
	}
	if !strings.Contains(item.RequestSummary, "nginx") {
		t.Fatalf("expected non-sensitive request summary to remain visible: %s", item.RequestSummary)
	}
}
