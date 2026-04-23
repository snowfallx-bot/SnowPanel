package service

import (
	"context"
	"testing"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient"
)

type fakeFileServiceAgentClient struct {
	grpcclient.AgentClient

	listFilesFn    func(context.Context, grpcclient.ListFilesRequest) (grpcclient.ListFilesResult, error)
	readTextFileFn func(context.Context, grpcclient.ReadTextFileRequest) (grpcclient.ReadTextFileResult, error)
	writeTextFn    func(context.Context, grpcclient.WriteTextFileRequest) (grpcclient.WriteTextFileResult, error)
	deleteFileFn   func(context.Context, grpcclient.DeleteFileRequest) (grpcclient.DeleteFileResult, error)
}

func (f *fakeFileServiceAgentClient) ListFiles(
	ctx context.Context,
	req grpcclient.ListFilesRequest,
) (grpcclient.ListFilesResult, error) {
	if f.listFilesFn != nil {
		return f.listFilesFn(ctx, req)
	}
	return grpcclient.ListFilesResult{}, nil
}

func (f *fakeFileServiceAgentClient) ReadTextFile(
	ctx context.Context,
	req grpcclient.ReadTextFileRequest,
) (grpcclient.ReadTextFileResult, error) {
	if f.readTextFileFn != nil {
		return f.readTextFileFn(ctx, req)
	}
	return grpcclient.ReadTextFileResult{}, nil
}

func (f *fakeFileServiceAgentClient) WriteTextFile(
	ctx context.Context,
	req grpcclient.WriteTextFileRequest,
) (grpcclient.WriteTextFileResult, error) {
	if f.writeTextFn != nil {
		return f.writeTextFn(ctx, req)
	}
	return grpcclient.WriteTextFileResult{}, nil
}

func (f *fakeFileServiceAgentClient) DeleteFile(
	ctx context.Context,
	req grpcclient.DeleteFileRequest,
) (grpcclient.DeleteFileResult, error) {
	if f.deleteFileFn != nil {
		return f.deleteFileFn(ctx, req)
	}
	return grpcclient.DeleteFileResult{}, nil
}

func TestFileServiceRenameFileSuccess(t *testing.T) {
	calls := make([]string, 0, 4)
	client := &fakeFileServiceAgentClient{
		listFilesFn: func(_ context.Context, req grpcclient.ListFilesRequest) (grpcclient.ListFilesResult, error) {
			calls = append(calls, "list:"+req.Path)
			return grpcclient.ListFilesResult{
				CurrentPath: "/tmp",
				Entries: []grpcclient.FileEntry{
					{Name: "a.txt", Path: "/tmp/a.txt"},
				},
			}, nil
		},
		readTextFileFn: func(_ context.Context, req grpcclient.ReadTextFileRequest) (grpcclient.ReadTextFileResult, error) {
			calls = append(calls, "read:"+req.Path)
			return grpcclient.ReadTextFileResult{
				Path:      req.Path,
				Content:   "hello",
				Size:      5,
				Truncated: false,
				Encoding:  "utf-8",
			}, nil
		},
		writeTextFn: func(_ context.Context, req grpcclient.WriteTextFileRequest) (grpcclient.WriteTextFileResult, error) {
			calls = append(calls, "write:"+req.Path)
			return grpcclient.WriteTextFileResult{
				Path:         req.Path,
				WrittenBytes: uint64(len(req.Content)),
			}, nil
		},
		deleteFileFn: func(_ context.Context, req grpcclient.DeleteFileRequest) (grpcclient.DeleteFileResult, error) {
			calls = append(calls, "delete:"+req.Path)
			return grpcclient.DeleteFileResult{Path: req.Path}, nil
		},
	}

	service := NewFileService(client)
	result, err := service.RenameFile(context.Background(), dto.RenameFileRequest{
		SourcePath: "/tmp/a.txt",
		TargetPath: "/tmp/b.txt",
	})
	if err != nil {
		t.Fatalf("expected rename success, got error: %v", err)
	}
	if result.SourcePath != "/tmp/a.txt" {
		t.Fatalf("unexpected source path: %s", result.SourcePath)
	}
	if result.TargetPath != "/tmp/b.txt" {
		t.Fatalf("unexpected target path: %s", result.TargetPath)
	}
	if result.WrittenBytes != 5 {
		t.Fatalf("unexpected written bytes: %d", result.WrittenBytes)
	}

	if len(calls) != 4 {
		t.Fatalf("unexpected call count: %d (%v)", len(calls), calls)
	}
	if calls[0] != "list:/tmp" || calls[1] != "read:/tmp/a.txt" || calls[2] != "write:/tmp/b.txt" || calls[3] != "delete:/tmp/a.txt" {
		t.Fatalf("unexpected call order: %v", calls)
	}
}

func TestFileServiceRenameFileTargetExists(t *testing.T) {
	readCalled := false
	writeCalled := false
	deleteCalled := false

	client := &fakeFileServiceAgentClient{
		listFilesFn: func(_ context.Context, _ grpcclient.ListFilesRequest) (grpcclient.ListFilesResult, error) {
			return grpcclient.ListFilesResult{
				CurrentPath: "/tmp",
				Entries: []grpcclient.FileEntry{
					{Name: "b.txt", Path: "/tmp/b.txt"},
				},
			}, nil
		},
		readTextFileFn: func(_ context.Context, _ grpcclient.ReadTextFileRequest) (grpcclient.ReadTextFileResult, error) {
			readCalled = true
			return grpcclient.ReadTextFileResult{}, nil
		},
		writeTextFn: func(_ context.Context, _ grpcclient.WriteTextFileRequest) (grpcclient.WriteTextFileResult, error) {
			writeCalled = true
			return grpcclient.WriteTextFileResult{}, nil
		},
		deleteFileFn: func(_ context.Context, _ grpcclient.DeleteFileRequest) (grpcclient.DeleteFileResult, error) {
			deleteCalled = true
			return grpcclient.DeleteFileResult{}, nil
		},
	}

	service := NewFileService(client)
	_, err := service.RenameFile(context.Background(), dto.RenameFileRequest{
		SourcePath: "/tmp/a.txt",
		TargetPath: "/tmp/b.txt",
	})
	if err == nil {
		t.Fatalf("expected rename error when target exists")
	}
	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("expected app error, got: %T", err)
	}
	if appErr.Code != apperror.ErrBadRequest.Code {
		t.Fatalf("expected bad request code, got: %d", appErr.Code)
	}
	if readCalled || writeCalled || deleteCalled {
		t.Fatalf("unexpected calls after target-exists validation: read=%v write=%v delete=%v", readCalled, writeCalled, deleteCalled)
	}
}

func TestFileServiceRenameFileRejectsTruncatedSource(t *testing.T) {
	writeCalled := false
	deleteCalled := false

	client := &fakeFileServiceAgentClient{
		listFilesFn: func(_ context.Context, _ grpcclient.ListFilesRequest) (grpcclient.ListFilesResult, error) {
			return grpcclient.ListFilesResult{
				CurrentPath: "/tmp",
				Entries: []grpcclient.FileEntry{
					{Name: "a.txt", Path: "/tmp/a.txt"},
				},
			}, nil
		},
		readTextFileFn: func(_ context.Context, req grpcclient.ReadTextFileRequest) (grpcclient.ReadTextFileResult, error) {
			return grpcclient.ReadTextFileResult{
				Path:      req.Path,
				Content:   "partial",
				Size:      16 * 1024 * 1024,
				Truncated: true,
				Encoding:  "utf-8",
			}, nil
		},
		writeTextFn: func(_ context.Context, _ grpcclient.WriteTextFileRequest) (grpcclient.WriteTextFileResult, error) {
			writeCalled = true
			return grpcclient.WriteTextFileResult{}, nil
		},
		deleteFileFn: func(_ context.Context, _ grpcclient.DeleteFileRequest) (grpcclient.DeleteFileResult, error) {
			deleteCalled = true
			return grpcclient.DeleteFileResult{}, nil
		},
	}

	service := NewFileService(client)
	_, err := service.RenameFile(context.Background(), dto.RenameFileRequest{
		SourcePath: "/tmp/a.txt",
		TargetPath: "/tmp/b.txt",
	})
	if err == nil {
		t.Fatalf("expected rename error when source content is truncated")
	}
	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("expected app error, got: %T", err)
	}
	if appErr.Code != apperror.ErrBadRequest.Code {
		t.Fatalf("expected bad request code, got: %d", appErr.Code)
	}
	if writeCalled || deleteCalled {
		t.Fatalf("unexpected calls after truncated read: write=%v delete=%v", writeCalled, deleteCalled)
	}
}

func TestFileServiceDownloadTextFileSuccess(t *testing.T) {
	var captured grpcclient.ReadTextFileRequest
	client := &fakeFileServiceAgentClient{
		readTextFileFn: func(
			_ context.Context,
			req grpcclient.ReadTextFileRequest,
		) (grpcclient.ReadTextFileResult, error) {
			captured = req
			return grpcclient.ReadTextFileResult{
				Path:      "/tmp/sample.log",
				Content:   "hello world",
				Size:      11,
				Truncated: false,
				Encoding:  "utf-8",
			}, nil
		},
	}

	service := NewFileService(client)
	result, err := service.DownloadTextFile(context.Background(), dto.DownloadFileQuery{
		Path: " /tmp/sample.log ",
	})
	if err != nil {
		t.Fatalf("expected download success, got error: %v", err)
	}
	if result.Path != "/tmp/sample.log" {
		t.Fatalf("unexpected path: %s", result.Path)
	}
	if result.Content != "hello world" {
		t.Fatalf("unexpected content: %s", result.Content)
	}
	if captured.Path != "/tmp/sample.log" {
		t.Fatalf("expected trimmed path, got %s", captured.Path)
	}
	if captured.MaxBytes != 8*1024*1024 {
		t.Fatalf("unexpected max bytes: %d", captured.MaxBytes)
	}
	if captured.Encoding != "utf-8" {
		t.Fatalf("unexpected encoding: %s", captured.Encoding)
	}
}

func TestFileServiceDownloadTextFileRejectsEmptyPath(t *testing.T) {
	client := &fakeFileServiceAgentClient{
		readTextFileFn: func(
			_ context.Context,
			_ grpcclient.ReadTextFileRequest,
		) (grpcclient.ReadTextFileResult, error) {
			t.Fatalf("read should not be called for empty path")
			return grpcclient.ReadTextFileResult{}, nil
		},
	}

	service := NewFileService(client)
	_, err := service.DownloadTextFile(context.Background(), dto.DownloadFileQuery{Path: "  "})
	if err == nil {
		t.Fatalf("expected bad request error")
	}
	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != apperror.ErrBadRequest.Code {
		t.Fatalf("expected bad request code, got %d", appErr.Code)
	}
}

func TestFileServiceDownloadTextFileRejectsTruncatedResult(t *testing.T) {
	client := &fakeFileServiceAgentClient{
		readTextFileFn: func(
			_ context.Context,
			req grpcclient.ReadTextFileRequest,
		) (grpcclient.ReadTextFileResult, error) {
			return grpcclient.ReadTextFileResult{
				Path:      req.Path,
				Content:   "partial",
				Size:      20 * 1024 * 1024,
				Truncated: true,
				Encoding:  "utf-8",
			}, nil
		},
	}

	service := NewFileService(client)
	_, err := service.DownloadTextFile(context.Background(), dto.DownloadFileQuery{
		Path: "/tmp/huge.log",
	})
	if err == nil {
		t.Fatalf("expected bad request error for truncated result")
	}
	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != apperror.ErrBadRequest.Code {
		t.Fatalf("expected bad request code, got %d", appErr.Code)
	}
}
