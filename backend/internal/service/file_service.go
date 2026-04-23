package service

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient"
)

type FileService interface {
	ListFiles(ctx context.Context, query dto.ListFilesQuery) (dto.ListFilesResult, error)
	ReadTextFile(ctx context.Context, req dto.ReadTextFileRequest) (dto.ReadTextFileResult, error)
	WriteTextFile(ctx context.Context, req dto.WriteTextFileRequest) (dto.WriteTextFileResult, error)
	CreateDirectory(ctx context.Context, req dto.CreateDirectoryRequest) (dto.CreateDirectoryResult, error)
	DeleteFile(ctx context.Context, req dto.DeleteFileRequest) (dto.DeleteFileResult, error)
	RenameFile(ctx context.Context, req dto.RenameFileRequest) (dto.RenameFileResult, error)
	DownloadFile(
		ctx context.Context,
		query dto.DownloadFileQuery,
		writeChunk func([]byte) error,
	) (dto.DownloadFileResult, error)
	UploadFile(
		ctx context.Context,
		req dto.UploadFileRequest,
		readChunk func([]byte) (int, error),
	) (dto.UploadFileResult, error)
}

type fileService struct {
	agentClient grpcclient.AgentClient
}

const downloadChunkSize uint32 = 256 * 1024
const uploadChunkSize = 256 * 1024

func NewFileService(agentClient grpcclient.AgentClient) FileService {
	return &fileService{agentClient: agentClient}
}

func (s *fileService) ListFiles(ctx context.Context, query dto.ListFilesQuery) (dto.ListFilesResult, error) {
	result, err := s.agentClient.ListFiles(ctx, grpcclient.ListFilesRequest{
		Path: query.Path,
	})
	if err != nil {
		return dto.ListFilesResult{}, mapAgentError(err)
	}

	entries := make([]dto.FileEntry, 0, len(result.Entries))
	for _, item := range result.Entries {
		entries = append(entries, dto.FileEntry{
			Name:       item.Name,
			Path:       item.Path,
			IsDir:      item.IsDir,
			Size:       item.Size,
			ModifiedAt: item.ModifiedAtUnix,
		})
	}

	return dto.ListFilesResult{
		CurrentPath: result.CurrentPath,
		Entries:     entries,
	}, nil
}

func (s *fileService) ReadTextFile(ctx context.Context, req dto.ReadTextFileRequest) (dto.ReadTextFileResult, error) {
	result, err := s.agentClient.ReadTextFile(ctx, grpcclient.ReadTextFileRequest{
		Path:     req.Path,
		MaxBytes: req.MaxBytes,
		Encoding: req.Encoding,
	})
	if err != nil {
		return dto.ReadTextFileResult{}, mapAgentError(err)
	}

	return dto.ReadTextFileResult{
		Path:      result.Path,
		Content:   result.Content,
		Size:      result.Size,
		Truncated: result.Truncated,
		Encoding:  result.Encoding,
	}, nil
}

func (s *fileService) WriteTextFile(ctx context.Context, req dto.WriteTextFileRequest) (dto.WriteTextFileResult, error) {
	result, err := s.agentClient.WriteTextFile(ctx, grpcclient.WriteTextFileRequest{
		Path:              req.Path,
		Content:           req.Content,
		CreateIfNotExists: req.CreateIfNotExists,
		Truncate:          req.Truncate,
		Encoding:          req.Encoding,
	})
	if err != nil {
		return dto.WriteTextFileResult{}, mapAgentError(err)
	}

	return dto.WriteTextFileResult{
		Path:         result.Path,
		WrittenBytes: result.WrittenBytes,
	}, nil
}

func (s *fileService) CreateDirectory(ctx context.Context, req dto.CreateDirectoryRequest) (dto.CreateDirectoryResult, error) {
	result, err := s.agentClient.CreateDirectory(ctx, grpcclient.CreateDirectoryRequest{
		Path:          req.Path,
		CreateParents: req.CreateParents,
	})
	if err != nil {
		return dto.CreateDirectoryResult{}, mapAgentError(err)
	}

	return dto.CreateDirectoryResult{Path: result.Path}, nil
}

func (s *fileService) DeleteFile(ctx context.Context, req dto.DeleteFileRequest) (dto.DeleteFileResult, error) {
	result, err := s.agentClient.DeleteFile(ctx, grpcclient.DeleteFileRequest{
		Path:      req.Path,
		Recursive: req.Recursive,
	})
	if err != nil {
		return dto.DeleteFileResult{}, mapAgentError(err)
	}

	return dto.DeleteFileResult{Path: result.Path}, nil
}

func (s *fileService) DownloadFile(
	ctx context.Context,
	query dto.DownloadFileQuery,
	writeChunk func([]byte) error,
) (dto.DownloadFileResult, error) {
	path := strings.TrimSpace(query.Path)
	if path == "" {
		return dto.DownloadFileResult{}, apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			fmt.Errorf("path is required"),
		)
	}
	if writeChunk == nil {
		return dto.DownloadFileResult{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			fmt.Errorf("download writer is required"),
		)
	}

	var (
		offset          uint64
		normalizedPath  string
		downloadedBytes uint64
		totalSize       uint64
	)

	for {
		chunk, err := s.agentClient.ReadFileChunk(ctx, grpcclient.ReadFileChunkRequest{
			Path:   path,
			Offset: offset,
			Limit:  downloadChunkSize,
		})
		if err != nil {
			return dto.DownloadFileResult{}, mapAgentError(err)
		}

		if normalizedPath == "" {
			normalizedPath = strings.TrimSpace(chunk.Path)
			if normalizedPath == "" {
				normalizedPath = path
			}
		}
		if chunk.TotalSize > 0 {
			totalSize = chunk.TotalSize
		}

		if chunk.Offset != offset {
			return dto.DownloadFileResult{}, apperror.Wrap(
				apperror.ErrInternal.Code,
				apperror.ErrInternal.HTTPStatus,
				apperror.ErrInternal.Message,
				fmt.Errorf("unexpected chunk offset: got %d, want %d", chunk.Offset, offset),
			)
		}

		if len(chunk.Chunk) > 0 {
			if err := writeChunk(chunk.Chunk); err != nil {
				return dto.DownloadFileResult{}, apperror.Wrap(
					apperror.ErrInternal.Code,
					apperror.ErrInternal.HTTPStatus,
					"stream write failed",
					err,
				)
			}
			offset += uint64(len(chunk.Chunk))
			downloadedBytes += uint64(len(chunk.Chunk))
		}

		if chunk.EOF {
			break
		}
		if len(chunk.Chunk) == 0 {
			return dto.DownloadFileResult{}, apperror.Wrap(
				apperror.ErrInternal.Code,
				apperror.ErrInternal.HTTPStatus,
				apperror.ErrInternal.Message,
				fmt.Errorf("core-agent returned empty chunk before EOF"),
			)
		}
	}

	if totalSize == 0 {
		totalSize = downloadedBytes
	}

	return dto.DownloadFileResult{
		Path:            normalizedPath,
		TotalSize:       totalSize,
		DownloadedBytes: downloadedBytes,
	}, nil
}

func (s *fileService) UploadFile(
	ctx context.Context,
	req dto.UploadFileRequest,
	readChunk func([]byte) (int, error),
) (dto.UploadFileResult, error) {
	path := strings.TrimSpace(req.Path)
	if path == "" {
		return dto.UploadFileResult{}, apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			fmt.Errorf("path is required"),
		)
	}
	if readChunk == nil {
		return dto.UploadFileResult{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			fmt.Errorf("upload reader is required"),
		)
	}

	buffer := make([]byte, uploadChunkSize)
	var (
		offset         uint64
		normalizedPath string
		totalSize      uint64
	)

	for {
		readLen, readErr := readChunk(buffer)
		if readErr != nil && readErr != io.EOF {
			return dto.UploadFileResult{}, apperror.Wrap(
				apperror.ErrInternal.Code,
				apperror.ErrInternal.HTTPStatus,
				"stream read failed",
				readErr,
			)
		}

		if readLen < 0 || readLen > len(buffer) {
			return dto.UploadFileResult{}, apperror.Wrap(
				apperror.ErrInternal.Code,
				apperror.ErrInternal.HTTPStatus,
				apperror.ErrInternal.Message,
				fmt.Errorf("invalid stream chunk length: %d", readLen),
			)
		}

		if readLen > 0 {
			chunk := append([]byte(nil), buffer[:readLen]...)
			result, err := s.agentClient.WriteFileChunk(ctx, grpcclient.WriteFileChunkRequest{
				Path:              path,
				Offset:            offset,
				Chunk:             chunk,
				CreateIfNotExists: true,
				Truncate:          offset == 0,
			})
			if err != nil {
				return dto.UploadFileResult{}, mapAgentError(err)
			}

			if result.Offset != offset {
				return dto.UploadFileResult{}, apperror.Wrap(
					apperror.ErrInternal.Code,
					apperror.ErrInternal.HTTPStatus,
					apperror.ErrInternal.Message,
					fmt.Errorf("unexpected write offset: got %d, want %d", result.Offset, offset),
				)
			}

			if result.WrittenBytes != uint64(readLen) {
				return dto.UploadFileResult{}, apperror.Wrap(
					apperror.ErrInternal.Code,
					apperror.ErrInternal.HTTPStatus,
					apperror.ErrInternal.Message,
					fmt.Errorf("unexpected written bytes: got %d, want %d", result.WrittenBytes, readLen),
				)
			}

			offset += uint64(readLen)
			totalSize = result.TotalSize
			if normalizedPath == "" {
				normalizedPath = strings.TrimSpace(result.Path)
			}
		}

		if readErr == io.EOF {
			break
		}
	}

	if offset == 0 {
		result, err := s.agentClient.WriteFileChunk(ctx, grpcclient.WriteFileChunkRequest{
			Path:              path,
			Offset:            0,
			Chunk:             nil,
			CreateIfNotExists: true,
			Truncate:          true,
		})
		if err != nil {
			return dto.UploadFileResult{}, mapAgentError(err)
		}
		totalSize = result.TotalSize
		if normalizedPath == "" {
			normalizedPath = strings.TrimSpace(result.Path)
		}
	}

	if normalizedPath == "" {
		normalizedPath = path
	}
	if totalSize == 0 {
		totalSize = offset
	}

	return dto.UploadFileResult{
		Path:          normalizedPath,
		UploadedBytes: offset,
		TotalSize:     totalSize,
	}, nil
}

func (s *fileService) RenameFile(
	ctx context.Context,
	req dto.RenameFileRequest,
) (dto.RenameFileResult, error) {
	sourcePath := strings.TrimSpace(req.SourcePath)
	targetPath := strings.TrimSpace(req.TargetPath)
	if sourcePath == "" || targetPath == "" {
		return dto.RenameFileResult{}, apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			fmt.Errorf("source_path and target_path are required"),
		)
	}
	if sourcePath == targetPath {
		return dto.RenameFileResult{}, apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			fmt.Errorf("source_path and target_path cannot be the same"),
		)
	}

	result, err := s.agentClient.RenameFile(ctx, grpcclient.RenameFileRequest{
		SourcePath: sourcePath,
		TargetPath: targetPath,
	})
	if err != nil {
		return dto.RenameFileResult{}, mapAgentError(err)
	}

	normalizedSource := strings.TrimSpace(result.SourcePath)
	if normalizedSource == "" {
		normalizedSource = sourcePath
	}

	normalizedTarget := strings.TrimSpace(result.TargetPath)
	if normalizedTarget == "" {
		normalizedTarget = targetPath
	}

	return dto.RenameFileResult{
		SourcePath:   normalizedSource,
		TargetPath:   normalizedTarget,
		WrittenBytes: result.MovedBytes,
	}, nil
}
