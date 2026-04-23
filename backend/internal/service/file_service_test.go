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
	readChunkFn    func(context.Context, grpcclient.ReadFileChunkRequest) (grpcclient.ReadFileChunkResult, error)
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

func (f *fakeFileServiceAgentClient) ReadFileChunk(
	ctx context.Context,
	req grpcclient.ReadFileChunkRequest,
) (grpcclient.ReadFileChunkResult, error) {
	if f.readChunkFn != nil {
		return f.readChunkFn(ctx, req)
	}
	return grpcclient.ReadFileChunkResult{}, nil
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

func TestFileServiceDownloadFileSuccess(t *testing.T) {
	requests := make([]grpcclient.ReadFileChunkRequest, 0, 2)
	client := &fakeFileServiceAgentClient{
		readChunkFn: func(
			_ context.Context,
			req grpcclient.ReadFileChunkRequest,
		) (grpcclient.ReadFileChunkResult, error) {
			requests = append(requests, req)
			switch req.Offset {
			case 0:
				return grpcclient.ReadFileChunkResult{
					Path:      "/tmp/sample.bin",
					Offset:    0,
					Chunk:     []byte("hello "),
					TotalSize: 11,
					EOF:       false,
				}, nil
			case 6:
				return grpcclient.ReadFileChunkResult{
					Path:      "/tmp/sample.bin",
					Offset:    6,
					Chunk:     []byte("world"),
					TotalSize: 11,
					EOF:       true,
				}, nil
			default:
				t.Fatalf("unexpected offset %d", req.Offset)
				return grpcclient.ReadFileChunkResult{}, nil
			}
		},
	}

	chunks := make([][]byte, 0, 2)
	service := NewFileService(client)
	result, err := service.DownloadFile(
		context.Background(),
		dto.DownloadFileQuery{Path: " /tmp/sample.bin "},
		func(chunk []byte) error {
			copied := append([]byte(nil), chunk...)
			chunks = append(chunks, copied)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("expected download success, got error: %v", err)
	}
	if result.Path != "/tmp/sample.bin" {
		t.Fatalf("unexpected path: %s", result.Path)
	}
	if result.TotalSize != 11 {
		t.Fatalf("unexpected total size: %d", result.TotalSize)
	}
	if result.DownloadedBytes != 11 {
		t.Fatalf("unexpected downloaded bytes: %d", result.DownloadedBytes)
	}
	if len(requests) != 2 {
		t.Fatalf("unexpected request count: %d", len(requests))
	}
	if requests[0].Path != "/tmp/sample.bin" || requests[0].Offset != 0 || requests[0].Limit != downloadChunkSize {
		t.Fatalf("unexpected first request: %+v", requests[0])
	}
	if requests[1].Path != "/tmp/sample.bin" || requests[1].Offset != 6 || requests[1].Limit != downloadChunkSize {
		t.Fatalf("unexpected second request: %+v", requests[1])
	}
	if len(chunks) != 2 {
		t.Fatalf("unexpected chunk count: %d", len(chunks))
	}
	if string(chunks[0]) != "hello " || string(chunks[1]) != "world" {
		t.Fatalf("unexpected chunk payloads: %q %q", string(chunks[0]), string(chunks[1]))
	}
}

func TestFileServiceDownloadFileRejectsEmptyPath(t *testing.T) {
	client := &fakeFileServiceAgentClient{
		readChunkFn: func(
			_ context.Context,
			_ grpcclient.ReadFileChunkRequest,
		) (grpcclient.ReadFileChunkResult, error) {
			t.Fatalf("read chunk should not be called for empty path")
			return grpcclient.ReadFileChunkResult{}, nil
		},
	}

	service := NewFileService(client)
	_, err := service.DownloadFile(
		context.Background(),
		dto.DownloadFileQuery{Path: "  "},
		func(_ []byte) error { return nil },
	)
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

func TestFileServiceDownloadFileRejectsUnexpectedOffset(t *testing.T) {
	client := &fakeFileServiceAgentClient{
		readChunkFn: func(
			_ context.Context,
			req grpcclient.ReadFileChunkRequest,
		) (grpcclient.ReadFileChunkResult, error) {
			return grpcclient.ReadFileChunkResult{
				Path:      req.Path,
				Offset:    7,
				Chunk:     []byte("chunk"),
				TotalSize: 32,
				EOF:       true,
			}, nil
		},
	}

	service := NewFileService(client)
	_, err := service.DownloadFile(
		context.Background(),
		dto.DownloadFileQuery{Path: "/tmp/sample.bin"},
		func(_ []byte) error { return nil },
	)
	if err == nil {
		t.Fatalf("expected internal error for unexpected chunk offset")
	}
	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != apperror.ErrInternal.Code {
		t.Fatalf("expected internal code, got %d", appErr.Code)
	}
}

func TestFileServiceDownloadFileRejectsEmptyNonEOFChunk(t *testing.T) {
	client := &fakeFileServiceAgentClient{
		readChunkFn: func(
			_ context.Context,
			req grpcclient.ReadFileChunkRequest,
		) (grpcclient.ReadFileChunkResult, error) {
			return grpcclient.ReadFileChunkResult{
				Path:      req.Path,
				Offset:    req.Offset,
				Chunk:     []byte{},
				TotalSize: 12,
				EOF:       false,
			}, nil
		},
	}

	service := NewFileService(client)
	_, err := service.DownloadFile(
		context.Background(),
		dto.DownloadFileQuery{Path: "/tmp/sample.bin"},
		func(_ []byte) error { return nil },
	)
	if err == nil {
		t.Fatalf("expected internal error for empty non-EOF chunk")
	}
	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != apperror.ErrInternal.Code {
		t.Fatalf("expected internal code, got %d", appErr.Code)
	}
}
