package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
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
	downloadChunks [][]byte
	downloadResult dto.DownloadFileResult
	downloadErr    error
	uploadReq      dto.UploadFileRequest
	uploadData     []byte
	uploadResult   dto.UploadFileResult
	uploadErr      error
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

func (s *fileHandlerServiceStub) DownloadFile(
	_ context.Context,
	query dto.DownloadFileQuery,
	writeChunk func([]byte) error,
) (dto.DownloadFileResult, error) {
	s.downloadQuery = query
	if s.downloadErr != nil {
		return dto.DownloadFileResult{}, s.downloadErr
	}
	for _, chunk := range s.downloadChunks {
		if err := writeChunk(chunk); err != nil {
			return dto.DownloadFileResult{}, err
		}
	}
	return s.downloadResult, s.downloadErr
}

func (s *fileHandlerServiceStub) UploadFile(
	_ context.Context,
	req dto.UploadFileRequest,
	readChunk func([]byte) (int, error),
) (dto.UploadFileResult, error) {
	s.uploadReq = req
	s.uploadData = nil
	if s.uploadErr != nil {
		return dto.UploadFileResult{}, s.uploadErr
	}

	buffer := make([]byte, 8)
	for {
		n, err := readChunk(buffer)
		if n > 0 {
			s.uploadData = append(s.uploadData, buffer[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return dto.UploadFileResult{}, err
		}
	}
	return s.uploadResult, nil
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
		downloadChunks: [][]byte{
			[]byte("hello "),
			[]byte("from file"),
		},
		downloadResult: dto.DownloadFileResult{
			Path:            "/tmp/sample.log",
			TotalSize:       15,
			DownloadedBytes: 15,
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

func TestFileHandlerDownloadFilePartialSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fileSvc := &fileHandlerServiceStub{
		downloadChunks: [][]byte{
			[]byte("from file"),
		},
		downloadResult: dto.DownloadFileResult{
			Path:            "/tmp/sample.log",
			StartOffset:     6,
			EndOffset:       14,
			TotalSize:       15,
			DownloadedBytes: 9,
		},
	}
	auditSvc := &fileAuditRecorder{}
	handler := NewFileHandler(fileSvc, auditSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/v1/files/download?path=/tmp/sample.log&offset=6",
		nil,
	)

	handler.DownloadFile(c)

	if w.Code != http.StatusPartialContent {
		t.Fatalf("expected 206, got %d", w.Code)
	}
	if got := w.Header().Get("Content-Range"); got != "bytes 6-14/15" {
		t.Fatalf("unexpected content-range: %s", got)
	}
	if got := w.Header().Get("Accept-Ranges"); got != "bytes" {
		t.Fatalf("unexpected accept-ranges: %s", got)
	}
	if body := w.Body.String(); body != "from file" {
		t.Fatalf("unexpected response body: %q", body)
	}
	if fileSvc.downloadQuery.Offset != 6 {
		t.Fatalf("unexpected download offset: %d", fileSvc.downloadQuery.Offset)
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

func TestFileHandlerUploadFileSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fileSvc := &fileHandlerServiceStub{
		uploadResult: dto.UploadFileResult{
			Path:          "/tmp/demo.bin",
			UploadedBytes: 11,
			TotalSize:     11,
		},
	}
	auditSvc := &fileAuditRecorder{}
	handler := NewFileHandler(fileSvc, auditSvc)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("path", "/tmp/demo.bin"); err != nil {
		t.Fatalf("write field failed: %v", err)
	}
	if err := writer.WriteField("offset", "5"); err != nil {
		t.Fatalf("write offset field failed: %v", err)
	}
	part, err := writer.CreateFormFile("file", "demo.bin")
	if err != nil {
		t.Fatalf("create form file failed: %v", err)
	}
	if _, err := part.Write([]byte("hello world")); err != nil {
		t.Fatalf("write file part failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer failed: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	handler.UploadFile(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if fileSvc.uploadReq.Path != "/tmp/demo.bin" {
		t.Fatalf("unexpected upload path: %s", fileSvc.uploadReq.Path)
	}
	if fileSvc.uploadReq.Offset != 5 {
		t.Fatalf("unexpected upload offset: %d", fileSvc.uploadReq.Offset)
	}
	if string(fileSvc.uploadData) != "hello world" {
		t.Fatalf("unexpected uploaded bytes: %q", string(fileSvc.uploadData))
	}
	if len(auditSvc.records) != 1 {
		t.Fatalf("expected one audit record, got %d", len(auditSvc.records))
	}
	record := auditSvc.records[0]
	if !record.Success || record.Action != "upload" || record.Module != "files" {
		t.Fatalf("unexpected upload audit record: %+v", record)
	}
}

func TestFileHandlerUploadFileRequiresFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fileSvc := &fileHandlerServiceStub{}
	auditSvc := &fileAuditRecorder{}
	handler := NewFileHandler(fileSvc, auditSvc)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("path", "/tmp/demo.bin"); err != nil {
		t.Fatalf("write field failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer failed: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	handler.UploadFile(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if len(auditSvc.records) != 0 {
		t.Fatalf("expected no audit record for validation failure, got %d", len(auditSvc.records))
	}
}
