package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/middleware"
)

type backupHandlerServiceStub struct {
	listQuery    dto.ListBackupsQuery
	listResult   dto.ListBackupsResult
	listErr      error
	createReq    dto.CreateBackupMetadataRequest
	createUserID *int64
	createResult dto.BackupSummary
	createErr    error
	verifyID     int64
	verifyReq    dto.VerifyBackupRequest
	verifyResult dto.BackupSummary
	verifyErr    error
	cleanupReq   dto.BackupRetentionCleanupRequest
	cleanupRes   dto.BackupRetentionCleanupResult
	cleanupErr   error
}

func (s *backupHandlerServiceStub) CreateMetadata(
	_ context.Context,
	req dto.CreateBackupMetadataRequest,
	createdBy *int64,
) (dto.BackupSummary, error) {
	s.createReq = req
	s.createUserID = createdBy
	return s.createResult, s.createErr
}

func (s *backupHandlerServiceStub) CreateArtifact(
	context.Context,
	int64,
) (dto.BackupSummary, error) {
	return dto.BackupSummary{}, errors.New("not implemented")
}

func (s *backupHandlerServiceStub) Verify(
	_ context.Context,
	id int64,
	req dto.VerifyBackupRequest,
) (dto.BackupSummary, error) {
	s.verifyID = id
	s.verifyReq = req
	return s.verifyResult, s.verifyErr
}

func (s *backupHandlerServiceStub) VerifyArtifact(
	context.Context,
	int64,
) (dto.BackupSummary, error) {
	return dto.BackupSummary{}, errors.New("not implemented")
}

func (s *backupHandlerServiceStub) MarkStatus(
	context.Context,
	int64,
	string,
) (dto.BackupSummary, error) {
	return dto.BackupSummary{}, errors.New("not implemented")
}

func (s *backupHandlerServiceStub) List(
	_ context.Context,
	query dto.ListBackupsQuery,
) (dto.ListBackupsResult, error) {
	s.listQuery = query
	return s.listResult, s.listErr
}

func (s *backupHandlerServiceStub) CleanupRetention(
	_ context.Context,
	req dto.BackupRetentionCleanupRequest,
) (dto.BackupRetentionCleanupResult, error) {
	s.cleanupReq = req
	return s.cleanupRes, s.cleanupErr
}

type backupAuditRecorder struct {
	records []dto.RecordAuditInput
}

func (s *backupAuditRecorder) Record(_ context.Context, input dto.RecordAuditInput) {
	s.records = append(s.records, input)
}

func (s *backupAuditRecorder) List(
	context.Context,
	dto.ListAuditLogsQuery,
) (dto.ListAuditLogsResult, error) {
	return dto.ListAuditLogsResult{}, errors.New("not implemented")
}

func (s *backupAuditRecorder) Export(
	context.Context,
	dto.ListAuditLogsQuery,
	string,
	io.Writer,
) error {
	return errors.New("not implemented")
}

func (s *backupAuditRecorder) CleanupRetention(
	context.Context,
	dto.AuditRetentionCleanupRequest,
) (dto.AuditRetentionCleanupResult, error) {
	return dto.AuditRetentionCleanupResult{}, errors.New("not implemented")
}

func TestBackupHandlerListPassesFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	backupSvc := &backupHandlerServiceStub{
		listResult: dto.ListBackupsResult{
			Page:  2,
			Size:  10,
			Total: 1,
			Items: []dto.BackupSummary{{ID: 8, Status: "success"}},
		},
	}
	handler := NewBackupHandler(backupSvc, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/v1/backups?page=2&size=10&status=success&resource_type=postgres&resource_id=primary",
		nil,
	)

	handler.ListBackups(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if backupSvc.listQuery.Page != 2 || backupSvc.listQuery.Size != 10 {
		t.Fatalf("unexpected pagination query: %+v", backupSvc.listQuery)
	}
	if backupSvc.listQuery.Status != "success" ||
		backupSvc.listQuery.ResourceType != "postgres" ||
		backupSvc.listQuery.ResourceID != "primary" {
		t.Fatalf("unexpected filters: %+v", backupSvc.listQuery)
	}
}

func TestBackupHandlerCreateRecordsAudit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now().Format(time.RFC3339)
	backupSvc := &backupHandlerServiceStub{
		createResult: dto.BackupSummary{
			ID:           12,
			ResourceType: "postgres",
			ResourceID:   "primary",
			StorageType:  "local",
			Status:       "pending",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}
	auditSvc := &backupAuditRecorder{}
	handler := NewBackupHandler(backupSvc, nil, auditSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/backups",
		strings.NewReader(`{"resource_type":"postgres","resource_id":"primary"}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")
	userID := int64(7)
	c.Set(middleware.CurrentUserIDKey, userID)
	c.Set(middleware.CurrentUsernameKey, "operator")

	handler.CreateBackup(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if backupSvc.createReq.ResourceType != "postgres" || backupSvc.createReq.ResourceID != "primary" {
		t.Fatalf("unexpected create request: %+v", backupSvc.createReq)
	}
	if backupSvc.createUserID == nil || *backupSvc.createUserID != userID {
		t.Fatalf("expected user id to be propagated")
	}
	if len(auditSvc.records) != 1 {
		t.Fatalf("expected one audit record, got %d", len(auditSvc.records))
	}
	record := auditSvc.records[0]
	if record.Module != "backups" || record.Action != "create" || !record.Success {
		t.Fatalf("unexpected audit record: %+v", record)
	}
	if record.TargetID != "12" {
		t.Fatalf("expected backup id target, got %q", record.TargetID)
	}

	body := map[string]any{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if body["code"].(float64) != 0 {
		t.Fatalf("expected ok envelope, got %+v", body)
	}
}

func TestBackupHandlerVerifyFailureRecordsAudit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	backupSvc := &backupHandlerServiceStub{
		verifyErr: apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			errors.New("backup checksum mismatch"),
		),
	}
	auditSvc := &backupAuditRecorder{}
	handler := NewBackupHandler(backupSvc, nil, auditSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/backups/12/verify",
		strings.NewReader(`{"size_bytes":2048,"checksum":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "12"}}

	handler.VerifyBackup(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if backupSvc.verifyID != 12 || backupSvc.verifyReq.SizeBytes != 2048 {
		t.Fatalf("unexpected verify args: id=%d req=%+v", backupSvc.verifyID, backupSvc.verifyReq)
	}
	if len(auditSvc.records) != 1 {
		t.Fatalf("expected one audit record, got %d", len(auditSvc.records))
	}
	record := auditSvc.records[0]
	if record.Success || record.Module != "backups" || record.Action != "verify" || record.TargetID != "12" {
		t.Fatalf("unexpected audit record: %+v", record)
	}
}

func TestBackupHandlerCleanupRetentionRecordsAudit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	backupSvc := &backupHandlerServiceStub{
		cleanupRes: dto.BackupRetentionCleanupResult{
			DryRun:        true,
			RetentionDays: 30,
			MatchedRows:   2,
		},
	}
	auditSvc := &backupAuditRecorder{}
	handler := NewBackupHandler(backupSvc, nil, auditSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/backups/retention/cleanup",
		strings.NewReader(`{"dry_run":true,"retention_days":30}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CleanupRetention(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !backupSvc.cleanupReq.DryRun || backupSvc.cleanupReq.RetentionDays != 30 {
		t.Fatalf("unexpected cleanup request: %+v", backupSvc.cleanupReq)
	}
	if len(auditSvc.records) != 1 {
		t.Fatalf("expected one audit record, got %d", len(auditSvc.records))
	}
	record := auditSvc.records[0]
	if record.Module != "backups" || record.Action != "retention_cleanup" || !record.Success {
		t.Fatalf("unexpected audit record: %+v", record)
	}
}
