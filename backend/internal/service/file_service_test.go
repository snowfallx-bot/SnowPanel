package service

import (
	"bytes"
	"context"
	"io"
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
	writeChunkFn   func(context.Context, grpcclient.WriteFileChunkRequest) (grpcclient.WriteFileChunkResult, error)
	writeTextFn    func(context.Context, grpcclient.WriteTextFileRequest) (grpcclient.WriteTextFileResult, error)
	deleteFileFn   func(context.Context, grpcclient.DeleteFileRequest) (grpcclient.DeleteFileResult, error)
	renameFileFn   func(context.Context, grpcclient.RenameFileRequest) (grpcclient.RenameFileResult, error)
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

func (f *fakeFileServiceAgentClient) WriteFileChunk(
	ctx context.Context,
	req grpcclient.WriteFileChunkRequest,
) (grpcclient.WriteFileChunkResult, error) {
	if f.writeChunkFn != nil {
		return f.writeChunkFn(ctx, req)
	}
	return grpcclient.WriteFileChunkResult{}, nil
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

func (f *fakeFileServiceAgentClient) RenameFile(
	ctx context.Context,
	req grpcclient.RenameFileRequest,
) (grpcclient.RenameFileResult, error) {
	if f.renameFileFn != nil {
		return f.renameFileFn(ctx, req)
	}
	return grpcclient.RenameFileResult{}, nil
}

func TestFileServiceRenameFileSuccess(t *testing.T) {
	calls := make([]string, 0, 1)
	client := &fakeFileServiceAgentClient{
		renameFileFn: func(_ context.Context, req grpcclient.RenameFileRequest) (grpcclient.RenameFileResult, error) {
			calls = append(calls, "rename:"+req.SourcePath+"->"+req.TargetPath)
			return grpcclient.RenameFileResult{
				SourcePath: req.SourcePath,
				TargetPath: req.TargetPath,
				MovedBytes: 5,
			}, nil
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
		t.Fatalf("unexpected moved bytes: %d", result.WrittenBytes)
	}

	if len(calls) != 1 {
		t.Fatalf("unexpected call count: %d (%v)", len(calls), calls)
	}
	if calls[0] != "rename:/tmp/a.txt->/tmp/b.txt" {
		t.Fatalf("unexpected call order: %v", calls)
	}
}

func TestFileServiceRenameFileTargetExists(t *testing.T) {
	renameCalled := false

	client := &fakeFileServiceAgentClient{
		renameFileFn: func(_ context.Context, _ grpcclient.RenameFileRequest) (grpcclient.RenameFileResult, error) {
			renameCalled = true
			return grpcclient.RenameFileResult{}, &grpcclient.AgentError{
				Code:    4000,
				Message: "bad request",
				Detail:  "target '/tmp/b.txt' already exists",
			}
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
	if appErr.Code != 4000 {
		t.Fatalf("expected mapped agent code 4000, got: %d", appErr.Code)
	}
	if !renameCalled {
		t.Fatalf("expected rename RPC to be called")
	}
}

func TestFileServiceRenameFileRejectsSamePath(t *testing.T) {
	renameCalled := false
	client := &fakeFileServiceAgentClient{
		renameFileFn: func(_ context.Context, _ grpcclient.RenameFileRequest) (grpcclient.RenameFileResult, error) {
			renameCalled = true
			return grpcclient.RenameFileResult{}, nil
		},
	}

	service := NewFileService(client)
	_, err := service.RenameFile(context.Background(), dto.RenameFileRequest{
		SourcePath: "/tmp/a.txt",
		TargetPath: "/tmp/a.txt",
	})
	if err == nil {
		t.Fatalf("expected rename validation error when source and target are the same")
	}
	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("expected app error, got: %T", err)
	}
	if appErr.Code != apperror.ErrBadRequest.Code {
		t.Fatalf("expected bad request code, got: %d", appErr.Code)
	}
	if renameCalled {
		t.Fatalf("rename RPC should not be called when local validation fails")
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

func TestFileServiceDownloadFileWithOffsetAndLimit(t *testing.T) {
	requests := make([]grpcclient.ReadFileChunkRequest, 0, 1)
	client := &fakeFileServiceAgentClient{
		readChunkFn: func(
			_ context.Context,
			req grpcclient.ReadFileChunkRequest,
		) (grpcclient.ReadFileChunkResult, error) {
			requests = append(requests, req)
			return grpcclient.ReadFileChunkResult{
				Path:      "/tmp/sample.bin",
				Offset:    6,
				Chunk:     []byte("world"),
				TotalSize: 11,
				EOF:       true,
			}, nil
		},
	}

	chunks := make([][]byte, 0, 1)
	service := NewFileService(client)
	result, err := service.DownloadFile(
		context.Background(),
		dto.DownloadFileQuery{Path: "/tmp/sample.bin", Offset: 6, Limit: 5},
		func(chunk []byte) error {
			chunks = append(chunks, append([]byte(nil), chunk...))
			return nil
		},
	)
	if err != nil {
		t.Fatalf("expected partial download success, got error: %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("unexpected request count: %d", len(requests))
	}
	if requests[0].Offset != 6 || requests[0].Limit != 5 {
		t.Fatalf("unexpected partial request: %+v", requests[0])
	}
	if result.StartOffset != 6 || result.EndOffset != 10 {
		t.Fatalf("unexpected offsets: start=%d end=%d", result.StartOffset, result.EndOffset)
	}
	if result.DownloadedBytes != 5 || result.TotalSize != 11 {
		t.Fatalf("unexpected partial download result: %+v", result)
	}
	if len(chunks) != 1 || string(chunks[0]) != "world" {
		t.Fatalf("unexpected chunk payloads: %+v", chunks)
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

func TestFileServiceUploadFileSuccess(t *testing.T) {
	requests := make([]grpcclient.WriteFileChunkRequest, 0, 1)
	client := &fakeFileServiceAgentClient{
		writeChunkFn: func(
			_ context.Context,
			req grpcclient.WriteFileChunkRequest,
		) (grpcclient.WriteFileChunkResult, error) {
			requests = append(requests, req)
			return grpcclient.WriteFileChunkResult{
				Path:         "/tmp/demo.bin",
				Offset:       req.Offset,
				WrittenBytes: uint64(len(req.Chunk)),
				TotalSize:    uint64(len(req.Chunk)),
			}, nil
		},
	}

	reader := bytes.NewBufferString("hello world")
	service := NewFileService(client)
	result, err := service.UploadFile(
		context.Background(),
		dto.UploadFileRequest{Path: " /tmp/demo.bin "},
		reader.Read,
	)
	if err != nil {
		t.Fatalf("expected upload success, got error: %v", err)
	}
	if result.Path != "/tmp/demo.bin" {
		t.Fatalf("unexpected path: %s", result.Path)
	}
	if result.UploadedBytes != 11 {
		t.Fatalf("unexpected uploaded bytes: %d", result.UploadedBytes)
	}
	if result.TotalSize != 11 {
		t.Fatalf("unexpected total size: %d", result.TotalSize)
	}
	if len(requests) != 1 {
		t.Fatalf("unexpected request count: %d", len(requests))
	}
	if requests[0].Path != "/tmp/demo.bin" {
		t.Fatalf("unexpected request path: %s", requests[0].Path)
	}
	if requests[0].Offset != 0 {
		t.Fatalf("unexpected request offset: %d", requests[0].Offset)
	}
	if !requests[0].CreateIfNotExists || !requests[0].Truncate {
		t.Fatalf("unexpected create/truncate flags: %+v", requests[0])
	}
	if string(requests[0].Chunk) != "hello world" {
		t.Fatalf("unexpected uploaded chunk: %q", string(requests[0].Chunk))
	}
}

func TestFileServiceUploadFileCreatesEmptyFile(t *testing.T) {
	requests := make([]grpcclient.WriteFileChunkRequest, 0, 1)
	client := &fakeFileServiceAgentClient{
		writeChunkFn: func(
			_ context.Context,
			req grpcclient.WriteFileChunkRequest,
		) (grpcclient.WriteFileChunkResult, error) {
			requests = append(requests, req)
			return grpcclient.WriteFileChunkResult{
				Path:         req.Path,
				Offset:       req.Offset,
				WrittenBytes: uint64(len(req.Chunk)),
				TotalSize:    0,
			}, nil
		},
	}

	reader := bytes.NewReader(nil)
	service := NewFileService(client)
	result, err := service.UploadFile(
		context.Background(),
		dto.UploadFileRequest{Path: "/tmp/empty.bin"},
		reader.Read,
	)
	if err != nil {
		t.Fatalf("expected empty upload success, got error: %v", err)
	}
	if result.Path != "/tmp/empty.bin" {
		t.Fatalf("unexpected path: %s", result.Path)
	}
	if result.UploadedBytes != 0 {
		t.Fatalf("unexpected uploaded bytes: %d", result.UploadedBytes)
	}
	if len(requests) != 1 {
		t.Fatalf("unexpected request count: %d", len(requests))
	}
	if requests[0].Offset != 0 || !requests[0].Truncate || !requests[0].CreateIfNotExists {
		t.Fatalf("unexpected empty-file write request: %+v", requests[0])
	}
	if len(requests[0].Chunk) != 0 {
		t.Fatalf("expected empty chunk for empty file, got %d bytes", len(requests[0].Chunk))
	}
}

func TestFileServiceUploadFileRejectsUnexpectedWriteOffset(t *testing.T) {
	client := &fakeFileServiceAgentClient{
		writeChunkFn: func(
			_ context.Context,
			req grpcclient.WriteFileChunkRequest,
		) (grpcclient.WriteFileChunkResult, error) {
			return grpcclient.WriteFileChunkResult{
				Path:         req.Path,
				Offset:       req.Offset + 1,
				WrittenBytes: uint64(len(req.Chunk)),
				TotalSize:    uint64(len(req.Chunk)),
			}, nil
		},
	}

	reader := bytes.NewBufferString("x")
	service := NewFileService(client)
	_, err := service.UploadFile(
		context.Background(),
		dto.UploadFileRequest{Path: "/tmp/demo.bin"},
		reader.Read,
	)
	if err == nil {
		t.Fatalf("expected internal error")
	}
	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != apperror.ErrInternal.Code {
		t.Fatalf("expected internal code, got %d", appErr.Code)
	}
}

func TestFileServiceUploadFileRejectsReadError(t *testing.T) {
	client := &fakeFileServiceAgentClient{
		writeChunkFn: func(
			_ context.Context,
			_ grpcclient.WriteFileChunkRequest,
		) (grpcclient.WriteFileChunkResult, error) {
			t.Fatalf("write chunk should not be called when reader fails")
			return grpcclient.WriteFileChunkResult{}, nil
		},
	}

	service := NewFileService(client)
	_, err := service.UploadFile(
		context.Background(),
		dto.UploadFileRequest{Path: "/tmp/demo.bin"},
		func(_ []byte) (int, error) {
			return 0, io.ErrUnexpectedEOF
		},
	)
	if err == nil {
		t.Fatalf("expected stream read error")
	}
	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != apperror.ErrInternal.Code {
		t.Fatalf("expected internal code, got %d", appErr.Code)
	}
}
