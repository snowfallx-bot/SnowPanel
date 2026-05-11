package service

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/repository"
)

type fakeAuditRepo struct {
	items         []model.AuditLog
	filters       []repository.AuditListFilter
	countBefore   int64
	deleteBefore  int64
	deletedCutoff *time.Time
}

func (r *fakeAuditRepo) Create(_ context.Context, item *model.AuditLog) error {
	cloned := *item
	r.items = append(r.items, cloned)
	return nil
}

func (r *fakeAuditRepo) List(
	_ context.Context,
	filter repository.AuditListFilter,
) ([]model.AuditLog, int64, error) {
	r.filters = append(r.filters, filter)
	if filter.Page > 1 {
		return nil, int64(len(r.items)), nil
	}
	return r.items, int64(len(r.items)), nil
}

func (r *fakeAuditRepo) CountBefore(_ context.Context, _ time.Time) (int64, error) {
	return r.countBefore, nil
}

func (r *fakeAuditRepo) DeleteBefore(_ context.Context, cutoff time.Time) (int64, error) {
	r.deletedCutoff = &cutoff
	return r.deleteBefore, nil
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
		RequestID:      "req-1",
		TraceID:        "trace-1",
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
	if item.RequestID != "req-1" || item.TraceID != "trace-1" {
		t.Fatalf("expected request/trace ids to be recorded, got request=%q trace=%q", item.RequestID, item.TraceID)
	}
}

func TestAuditListPassesForensicsFilters(t *testing.T) {
	repo := &fakeAuditRepo{}
	service := NewAuditService(repo)
	success := true

	_, err := service.List(context.Background(), dto.ListAuditLogsQuery{
		Page:       3,
		Size:       50,
		StartTime:  "2026-05-01T10:00:00Z",
		EndTime:    "2026-05-02",
		UserID:     7,
		Username:   "admin",
		Module:     "files",
		Action:     "delete",
		TargetType: "file",
		TargetID:   "/tmp/a.txt",
		Success:    &success,
		ResultCode: "ok",
		RequestID:  "req-123",
		TraceID:    "trace-456",
	})
	if err != nil {
		t.Fatalf("expected list to succeed, got %v", err)
	}
	if len(repo.filters) != 1 {
		t.Fatalf("expected one repository call, got %d", len(repo.filters))
	}
	filter := repo.filters[0]
	if filter.Page != 3 || filter.Size != 50 {
		t.Fatalf("unexpected pagination filter: %+v", filter)
	}
	if filter.StartTime == nil || !filter.StartTime.Equal(time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected start time: %v", filter.StartTime)
	}
	if filter.EndTime == nil || filter.EndTime.Before(time.Date(2026, 5, 2, 23, 59, 59, 0, time.UTC)) {
		t.Fatalf("unexpected end time: %v", filter.EndTime)
	}
	if filter.UserID == nil || *filter.UserID != 7 || filter.Success == nil || *filter.Success != true {
		t.Fatalf("unexpected user/success filters: %+v", filter)
	}
	if filter.Username != "admin" || filter.Module != "files" || filter.Action != "delete" ||
		filter.TargetType != "file" || filter.TargetID != "/tmp/a.txt" ||
		filter.ResultCode != "ok" || filter.RequestID != "req-123" || filter.TraceID != "trace-456" {
		t.Fatalf("unexpected forensic filters: %+v", filter)
	}
}

func TestAuditExportCSVRedactsAndPaginates(t *testing.T) {
	userID := int64(7)
	repo := &fakeAuditRepo{items: []model.AuditLog{
		{
			ID:             1,
			UserID:         &userID,
			Username:       "admin",
			IP:             "127.0.0.1",
			Module:         "settings",
			Action:         "update",
			TargetType:     "system_setting",
			TargetID:       "agent",
			RequestSummary: `{"token":"[REDACTED]","service_name":"nginx"}`,
			Success:        true,
			ResultCode:     "ok",
			ResultMessage:  "updated token=[REDACTED]",
			RequestID:      "req-1",
			TraceID:        "trace-1",
			CreatedAt:      time.Date(2026, 5, 11, 10, 0, 0, 0, time.UTC),
		},
	}}
	service := NewAuditService(repo)

	var output bytes.Buffer
	if err := service.Export(context.Background(), dto.ListAuditLogsQuery{}, "csv", &output); err != nil {
		t.Fatalf("expected csv export to succeed, got %v", err)
	}

	body := output.String()
	if !strings.Contains(body, "request_id,trace_id") || !strings.Contains(body, "req-1,trace-1") {
		t.Fatalf("expected csv export to include request and trace ids: %s", body)
	}
	if strings.Contains(body, "agent-token") || strings.Contains(body, "plain") {
		t.Fatalf("csv export leaked sensitive data: %s", body)
	}
}

func TestAuditExportJSONL(t *testing.T) {
	repo := &fakeAuditRepo{items: []model.AuditLog{
		{
			ID:             2,
			Username:       "operator",
			Module:         "docker",
			Action:         "restart",
			TargetType:     "container",
			TargetID:       "app",
			RequestSummary: `{}`,
			Success:        false,
			ResultCode:     "agent_unavailable",
			ResultMessage:  "core agent unavailable",
			RequestID:      "req-jsonl",
			TraceID:        "trace-jsonl",
			CreatedAt:      time.Date(2026, 5, 11, 10, 0, 0, 0, time.UTC),
		},
	}}
	service := NewAuditService(repo)

	var output bytes.Buffer
	if err := service.Export(context.Background(), dto.ListAuditLogsQuery{}, "jsonl", &output); err != nil {
		t.Fatalf("expected jsonl export to succeed, got %v", err)
	}

	body := output.String()
	if !strings.Contains(body, `"request_id":"req-jsonl"`) ||
		!strings.Contains(body, `"trace_id":"trace-jsonl"`) ||
		!strings.HasSuffix(body, "\n") {
		t.Fatalf("unexpected jsonl export body: %s", body)
	}
}

func TestAuditRetentionCleanupDryRunDoesNotDelete(t *testing.T) {
	repo := &fakeAuditRepo{countBefore: 12, deleteBefore: 12}
	service := NewAuditServiceWithOptions(repo, AuditServiceOptions{
		RetentionDays: 90,
		ExportMaxRows: 10,
	})

	result, err := service.CleanupRetention(context.Background(), dto.AuditRetentionCleanupRequest{
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("expected dry-run cleanup to succeed, got %v", err)
	}

	if !result.DryRun || result.RetentionDays != 90 || result.MatchedRows != 12 || result.DeletedRows != 0 {
		t.Fatalf("unexpected dry-run result: %+v", result)
	}
	if repo.deletedCutoff != nil {
		t.Fatalf("dry-run cleanup should not delete rows")
	}
}

func TestAuditRetentionCleanupDeletesWhenDryRunFalse(t *testing.T) {
	repo := &fakeAuditRepo{countBefore: 12, deleteBefore: 7}
	service := NewAuditServiceWithOptions(repo, AuditServiceOptions{
		RetentionDays: 180,
		ExportMaxRows: 10,
	})

	result, err := service.CleanupRetention(context.Background(), dto.AuditRetentionCleanupRequest{
		RetentionDays: 30,
	})
	if err != nil {
		t.Fatalf("expected cleanup to succeed, got %v", err)
	}

	if result.DryRun || result.RetentionDays != 30 || result.MatchedRows != 12 || result.DeletedRows != 7 {
		t.Fatalf("unexpected cleanup result: %+v", result)
	}
	if repo.deletedCutoff == nil {
		t.Fatalf("expected cleanup to delete rows")
	}
}
