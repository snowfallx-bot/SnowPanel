package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/repository"
)

const (
	BackupStatusPending = "pending"
	BackupStatusRunning = "running"
	BackupStatusSuccess = "success"
	BackupStatusFailed  = "failed"

	BackupStorageLocal = "local"

	BackupResourcePostgres          = "postgres"
	BackupResourceAppMetadata       = "app_metadata"
	BackupResourceObservability     = "observability_config"
	BackupResourceCoreAgentTemplate = "core_agent_config"
)

type BackupService interface {
	CreateMetadata(ctx context.Context, req dto.CreateBackupMetadataRequest, createdBy *int64) (dto.BackupSummary, error)
	Verify(ctx context.Context, id int64, req dto.VerifyBackupRequest) (dto.BackupSummary, error)
	List(ctx context.Context, query dto.ListBackupsQuery) (dto.ListBackupsResult, error)
}

type backupService struct {
	repo repository.BackupRepository
}

func NewBackupService(repo repository.BackupRepository) BackupService {
	return &backupService{repo: repo}
}

func (s *backupService) CreateMetadata(
	ctx context.Context,
	req dto.CreateBackupMetadataRequest,
	createdBy *int64,
) (dto.BackupSummary, error) {
	resourceType, err := normalizeBackupResourceType(req.ResourceType)
	if err != nil {
		return dto.BackupSummary{}, badBackupRequest(err)
	}
	resourceID, err := normalizeBackupText(req.ResourceID, "resource_id", 128, true)
	if err != nil {
		return dto.BackupSummary{}, badBackupRequest(err)
	}
	storageType, err := normalizeBackupStorageType(req.StorageType)
	if err != nil {
		return dto.BackupSummary{}, badBackupRequest(err)
	}
	filePath, err := normalizeBackupText(req.FilePath, "file_path", 1024, false)
	if err != nil {
		return dto.BackupSummary{}, badBackupRequest(err)
	}

	backup := &model.Backup{
		ResourceType: resourceType,
		ResourceID:   resourceID,
		StorageType:  storageType,
		FilePath:     filePath,
		SizeBytes:    0,
		Checksum:     "",
		Status:       BackupStatusPending,
		CreatedBy:    createdBy,
	}
	if err := s.repo.Create(ctx, backup); err != nil {
		return dto.BackupSummary{}, wrapBackupInternal(err)
	}
	return mapBackupSummary(*backup), nil
}

func (s *backupService) Verify(
	ctx context.Context,
	id int64,
	req dto.VerifyBackupRequest,
) (dto.BackupSummary, error) {
	if id <= 0 {
		return dto.BackupSummary{}, badBackupRequest(errors.New("backup id must be positive"))
	}
	if req.SizeBytes <= 0 {
		return dto.BackupSummary{}, badBackupRequest(errors.New("size_bytes must be positive"))
	}
	checksum, err := normalizeBackupChecksum(req.Checksum)
	if err != nil {
		return dto.BackupSummary{}, badBackupRequest(err)
	}
	filePath, err := normalizeBackupText(req.FilePath, "file_path", 1024, false)
	if err != nil {
		return dto.BackupSummary{}, badBackupRequest(err)
	}

	backup, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.BackupSummary{}, wrapBackupInternal(err)
	}
	if backup == nil {
		return dto.BackupSummary{}, apperror.ErrBackupNotFound
	}

	if backup.Checksum != "" && backup.Checksum != checksum {
		_ = s.repo.UpdateStatus(ctx, id, BackupStatusFailed)
		return dto.BackupSummary{}, badBackupRequest(errors.New("backup checksum mismatch"))
	}
	if backup.SizeBytes > 0 && backup.SizeBytes != req.SizeBytes {
		_ = s.repo.UpdateStatus(ctx, id, BackupStatusFailed)
		return dto.BackupSummary{}, badBackupRequest(errors.New("backup size mismatch"))
	}

	if err := s.repo.UpdateVerification(ctx, id, BackupStatusSuccess, req.SizeBytes, checksum, filePath); err != nil {
		return dto.BackupSummary{}, wrapBackupInternal(err)
	}

	backup.SizeBytes = req.SizeBytes
	backup.Checksum = checksum
	backup.Status = BackupStatusSuccess
	if filePath != "" {
		backup.FilePath = filePath
	}
	backup.UpdatedAt = time.Now()
	return mapBackupSummary(*backup), nil
}

func (s *backupService) List(
	ctx context.Context,
	query dto.ListBackupsQuery,
) (dto.ListBackupsResult, error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	size := query.Size
	if size <= 0 {
		size = 20
	}

	status, err := normalizeBackupStatusFilter(query.Status)
	if err != nil {
		return dto.ListBackupsResult{}, badBackupRequest(err)
	}
	resourceType := ""
	if strings.TrimSpace(query.ResourceType) != "" {
		resourceType, err = normalizeBackupResourceType(query.ResourceType)
		if err != nil {
			return dto.ListBackupsResult{}, badBackupRequest(err)
		}
	}
	resourceID := strings.TrimSpace(query.ResourceID)

	items, total, err := s.repo.List(ctx, repository.BackupListFilter{
		Page:         page,
		Size:         size,
		Status:       status,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	})
	if err != nil {
		return dto.ListBackupsResult{}, wrapBackupInternal(err)
	}

	result := make([]dto.BackupSummary, 0, len(items))
	for _, item := range items {
		result = append(result, mapBackupSummary(item))
	}
	return dto.ListBackupsResult{
		Page:  page,
		Size:  size,
		Total: total,
		Items: result,
	}, nil
}

func normalizeBackupResourceType(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case BackupResourcePostgres,
		BackupResourceAppMetadata,
		BackupResourceObservability,
		BackupResourceCoreAgentTemplate:
		return value, nil
	default:
		return "", fmt.Errorf("resource_type %q is outside the P3 backup scope", raw)
	}
}

func normalizeBackupStorageType(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return BackupStorageLocal, nil
	}
	if value != BackupStorageLocal {
		return "", errors.New("only local backup storage is supported in P3")
	}
	return value, nil
}

func normalizeBackupStatusFilter(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return "", nil
	}
	switch value {
	case BackupStatusPending, BackupStatusRunning, BackupStatusSuccess, BackupStatusFailed:
		return value, nil
	default:
		return "", fmt.Errorf("unsupported backup status %q", raw)
	}
}

func normalizeBackupText(raw string, field string, maxLength int, required bool) (string, error) {
	value := strings.TrimSpace(raw)
	if required && value == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	if len(value) > maxLength {
		return "", fmt.Errorf("%s must be at most %d bytes", field, maxLength)
	}
	if strings.ContainsFunc(value, unicode.IsControl) {
		return "", fmt.Errorf("%s cannot contain control characters", field)
	}
	return value, nil
}

func normalizeBackupChecksum(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.TrimPrefix(value, "sha256:")
	if len(value) != 64 {
		return "", errors.New("checksum must be a sha256 hex digest")
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return "", errors.New("checksum must be a sha256 hex digest")
		}
	}
	return "sha256:" + value, nil
}

func mapBackupSummary(backup model.Backup) dto.BackupSummary {
	return dto.BackupSummary{
		ID:           backup.ID,
		ResourceType: backup.ResourceType,
		ResourceID:   backup.ResourceID,
		StorageType:  backup.StorageType,
		FilePath:     backup.FilePath,
		SizeBytes:    backup.SizeBytes,
		Checksum:     backup.Checksum,
		Status:       backup.Status,
		CreatedBy:    backup.CreatedBy,
		CreatedAt:    backup.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    backup.UpdatedAt.Format(time.RFC3339),
	}
}

func badBackupRequest(err error) error {
	return apperror.Wrap(
		apperror.ErrBadRequest.Code,
		apperror.ErrBadRequest.HTTPStatus,
		apperror.ErrBadRequest.Message,
		err,
	)
}

func wrapBackupInternal(err error) error {
	return apperror.Wrap(
		apperror.ErrInternal.Code,
		http.StatusInternalServerError,
		apperror.ErrInternal.Message,
		err,
	)
}
