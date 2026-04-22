package service

import (
	"context"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient"
)

type FileService interface {
	ListFiles(ctx context.Context, query dto.ListFilesQuery) (dto.ListFilesResult, error)
	ReadTextFile(ctx context.Context, req dto.ReadTextFileRequest) (dto.ReadTextFileResult, error)
	WriteTextFile(ctx context.Context, req dto.WriteTextFileRequest) (dto.WriteTextFileResult, error)
	CreateDirectory(ctx context.Context, req dto.CreateDirectoryRequest) (dto.CreateDirectoryResult, error)
	DeleteFile(ctx context.Context, req dto.DeleteFileRequest) (dto.DeleteFileResult, error)
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

	s.emitAudit(ctx, "files", "list", query.Path, true)
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
		s.emitAudit(ctx, "files", "read", req.Path, false)
		return dto.ReadTextFileResult{}, mapAgentError(err)
	}

	s.emitAudit(ctx, "files", "read", req.Path, true)
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
		s.emitAudit(ctx, "files", "write", req.Path, false)
		return dto.WriteTextFileResult{}, mapAgentError(err)
	}

	s.emitAudit(ctx, "files", "write", req.Path, true)
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
		s.emitAudit(ctx, "files", "mkdir", req.Path, false)
		return dto.CreateDirectoryResult{}, mapAgentError(err)
	}

	s.emitAudit(ctx, "files", "mkdir", req.Path, true)
	return dto.CreateDirectoryResult{Path: result.Path}, nil
}

func (s *fileService) DeleteFile(ctx context.Context, req dto.DeleteFileRequest) (dto.DeleteFileResult, error) {
	result, err := s.agentClient.DeleteFile(ctx, grpcclient.DeleteFileRequest{
		Path:      req.Path,
		Recursive: req.Recursive,
	})
	if err != nil {
		s.emitAudit(ctx, "files", "delete", req.Path, false)
		return dto.DeleteFileResult{}, mapAgentError(err)
	}

	s.emitAudit(ctx, "files", "delete", req.Path, true)
	return dto.DeleteFileResult{Path: result.Path}, nil
}

func (s *fileService) emitAudit(_ context.Context, _ string, _ string, _ string, _ bool) {
	// Reserved for audit integration in stage 19.
}
