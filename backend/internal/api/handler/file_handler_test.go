package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
)

type fileHandlerServiceStub struct {
	downloadQuery  dto.DownloadFileQuery
	downloadResult dto.ReadTextFileResult
	downloadErr    error
}

func (s *fileHandlerServiceStub) ListFiles(context.Context, dto.ListFilesQuery) (dto.ListFilesResult, error) {
	return dto.ListFilesResult{}, errors.New("not implemented")
}

func (s *fileHandlerServiceStub) ReadTextFile(
	context.Context,
	dto.ReadTextFileRequest,
) (dto.ReadTextFileResult, error) {
	return dto.ReadTextFileResult{}, errors.New("not implemented")
}

func (s *fileHandlerServiceStub) WriteTextFile(
	context.Context,
	dto.WriteTextFileRequest,
) (dto.WriteTextFileResult, error) {
	return dto.WriteTextFileResult{}, errors.New("not implemented")
}

func (s *fileHandlerServiceStub) CreateDirectory(
	context.Context,
	dto.CreateDirectoryRequest,
) (dto.CreateDirectoryResult, error) {
	return dto.CreateDirectoryResult{}, errors.New("not implemented")
}

func (s *fileHandlerServiceStub) DeleteFile(
	context.Context,
	dto.DeleteFileRequest,
) (dto.DeleteFileResult, error) {
	return dto.DeleteFileResult{}, errors.New("not implemented")
}

func (s *fileHandlerServiceStub) RenameFile(
	context.Context,
	dto.RenameFileRequest,
) (dto.RenameFileResult, error) {
	return dto.RenameFileResult{}, errors.New("not implemented")
}

func (s *fileHandlerServiceStub) DownloadTextFile(
	_ context.Context,
	query dto.DownloadFileQuery,
) (dto.ReadTextFileResult, error) {
	s.downloadQuery = query
	return s.downloadResult, s.downloadErr
}

type fileAuditRecorder struct {
	records []dto.RecordAuditInput
}

func (s *fileAuditRecorder) Record(_ context.Context, input dto.RecordAuditInput) {
	s.records = append(s.records, input)
}

func (s *fileAuditRecorder) List(
	context.Context,
	dto.ListAuditLogsQuery,
) (dto.ListAuditLogsResult, error) {
	return dto.ListAuditLogsResult{}, errors.New("not implemented")
}

func TestFileHandlerDownloadFileSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fileSvc := &fileHandlerServiceStub{
		downloadResult: dto.ReadTextFileResult{
			Path:    "/tmp/sample.log",
			Content: "hello from file",
		},
	}
	auditSvc := &fileAuditRecorder{}
	handler := NewFileHandler(fileSvc, auditSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/v1/files/download?path=/tmp/sample.log",
		nil,
	)

	handler.DownloadFile(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if fileSvc.downloadQuery.Path != "/tmp/sample.log" {
		t.Fatalf("unexpected download path: %s", fileSvc.downloadQuery.Path)
	}
	if body := w.Body.String(); body != "hello from file" {
		t.Fatalf("unexpected response body: %q", body)
	}
	if got := w.Header().Get("Content-Type"); !strings.Contains(got, "text/plain") {
		t.Fatalf("expected text/plain content type, got %s", got)
	}
	if got := w.Header().Get("Content-Disposition"); !strings.Contains(got, `filename="sample.log"`) {
		t.Fatalf("unexpected content disposition: %s", got)
	}
	if len(auditSvc.records) != 1 {
		t.Fatalf("expected one audit record, got %d", len(auditSvc.records))
	}
	record := auditSvc.records[0]
	if !record.Success || record.Action != "download" || record.Module != "files" {
		t.Fatalf("unexpected success audit record: %+v", record)
	}
}

func TestFileHandlerDownloadFileFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fileSvc := &fileHandlerServiceStub{
		downloadErr: apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			errors.New("binary file is not supported"),
		),
	}
	auditSvc := &fileAuditRecorder{}
	handler := NewFileHandler(fileSvc, auditSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/v1/files/download?path=/tmp/binary.bin",
		nil,
	)

	handler.DownloadFile(c)

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
	if record.TargetID != "/tmp/binary.bin" {
		t.Fatalf("unexpected target id: %s", record.TargetID)
	}
}
