package service

import (
	"context"
	"fmt"
	"path"
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
}

type fileService struct {
	agentClient grpcclient.AgentClient
}

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

	if err := s.ensureTargetDoesNotExist(ctx, targetPath); err != nil {
		return dto.RenameFileResult{}, err
	}

	readResult, err := s.agentClient.ReadTextFile(ctx, grpcclient.ReadTextFileRequest{
		Path:     sourcePath,
		MaxBytes: 8 * 1024 * 1024,
		Encoding: "utf-8",
	})
	if err != nil {
		return dto.RenameFileResult{}, mapAgentError(err)
	}
	if readResult.Truncated {
		return dto.RenameFileResult{}, apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			fmt.Errorf("rename only supports files that can be fully read in one request"),
		)
	}

	writeResult, err := s.agentClient.WriteTextFile(ctx, grpcclient.WriteTextFileRequest{
		Path:              targetPath,
		Content:           readResult.Content,
		CreateIfNotExists: true,
		Truncate:          true,
		Encoding:          "utf-8",
	})
	if err != nil {
		return dto.RenameFileResult{}, mapAgentError(err)
	}

	_, err = s.agentClient.DeleteFile(ctx, grpcclient.DeleteFileRequest{
		Path:      sourcePath,
		Recursive: false,
	})
	if err != nil {
		return dto.RenameFileResult{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			"rename failed after write; source cleanup failed",
			err,
		)
	}

	return dto.RenameFileResult{
		SourcePath:   sourcePath,
		TargetPath:   writeResult.Path,
		WrittenBytes: writeResult.WrittenBytes,
	}, nil
}

func (s *fileService) ensureTargetDoesNotExist(ctx context.Context, targetPath string) error {
	parent := path.Dir(targetPath)
	if parent == "." {
		parent = "/"
	}

	listResult, err := s.agentClient.ListFiles(ctx, grpcclient.ListFilesRequest{Path: parent})
	if err != nil {
		return mapAgentError(err)
	}

	targetName := path.Base(targetPath)
	for _, entry := range listResult.Entries {
		if entry.Name == targetName {
			return apperror.Wrap(
				apperror.ErrBadRequest.Code,
				apperror.ErrBadRequest.HTTPStatus,
				apperror.ErrBadRequest.Message,
				fmt.Errorf("target path already exists: %s", targetPath),
			)
		}
	}
	return nil
}
