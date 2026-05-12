package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	CreateArtifact(ctx context.Context, id int64) (dto.BackupSummary, error)
	Verify(ctx context.Context, id int64, req dto.VerifyBackupRequest) (dto.BackupSummary, error)
	VerifyArtifact(ctx context.Context, id int64) (dto.BackupSummary, error)
	MarkStatus(ctx context.Context, id int64, status string) (dto.BackupSummary, error)
	List(ctx context.Context, query dto.ListBackupsQuery) (dto.ListBackupsResult, error)
	CleanupRetention(ctx context.Context, req dto.BackupRetentionCleanupRequest) (dto.BackupRetentionCleanupResult, error)
}

type backupService struct {
	repo          repository.BackupRepository
	retentionDays int
	localDir      string
}

func NewBackupService(repo repository.BackupRepository) BackupService {
	return NewBackupServiceWithOptions(repo, BackupServiceOptions{})
}

type BackupServiceOptions struct {
	RetentionDays int
	LocalDir      string
}

func NewBackupServiceWithOptions(repo repository.BackupRepository, options BackupServiceOptions) BackupService {
	retentionDays := options.RetentionDays
	if retentionDays <= 0 {
		retentionDays = 30
	}
	localDir := strings.TrimSpace(options.LocalDir)
	if localDir == "" {
		localDir = "var/backups"
	}
	return &backupService{
		repo:          repo,
		retentionDays: retentionDays,
		localDir:      localDir,
	}
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

type backupArtifactManifest struct {
	Version      int      `json:"version"`
	Kind         string   `json:"kind"`
	BackupID     int64    `json:"backup_id"`
	ResourceType string   `json:"resource_type"`
	ResourceID   string   `json:"resource_id"`
	StorageType  string   `json:"storage_type"`
	CreatedAt    string   `json:"created_at"`
	Scope        []string `json:"scope"`
	Note         string   `json:"note,omitempty"`
}

func (s *backupService) CreateArtifact(ctx context.Context, id int64) (dto.BackupSummary, error) {
	if id <= 0 {
		return dto.BackupSummary{}, badBackupRequest(errors.New("backup id must be positive"))
	}

	backup, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.BackupSummary{}, wrapBackupInternal(err)
	}
	if backup == nil {
		return dto.BackupSummary{}, apperror.ErrBackupNotFound
	}
	if backup.StorageType != BackupStorageLocal {
		_ = s.repo.UpdateStatus(ctx, id, BackupStatusFailed)
		return dto.BackupSummary{}, badBackupRequest(errors.New("only local backup storage can create artifacts in P3"))
	}

	if err := s.repo.UpdateStatus(ctx, id, BackupStatusRunning); err != nil {
		return dto.BackupSummary{}, wrapBackupInternal(err)
	}

	now := time.Now().UTC()
	path, err := safeBackupArtifactPath(s.localDir, *backup, now)
	if err != nil {
		_ = s.repo.UpdateStatus(ctx, id, BackupStatusFailed)
		return dto.BackupSummary{}, wrapBackupInternal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		_ = s.repo.UpdateStatus(ctx, id, BackupStatusFailed)
		return dto.BackupSummary{}, wrapBackupInternal(err)
	}

	manifest := backupArtifactManifest{
		Version:      1,
		Kind:         "snowpanel.backup.artifact",
		BackupID:     backup.ID,
		ResourceType: backup.ResourceType,
		ResourceID:   backup.ResourceID,
		StorageType:  backup.StorageType,
		CreatedAt:    now.Format(time.RFC3339),
		Scope:        backupArtifactScope(backup.ResourceType),
		Note:         backupArtifactNote(backup.ResourceType),
	}
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		_ = s.repo.UpdateStatus(ctx, id, BackupStatusFailed)
		return dto.BackupSummary{}, wrapBackupInternal(err)
	}
	content = append(content, '\n')

	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, content, 0600); err != nil {
		_ = s.repo.UpdateStatus(ctx, id, BackupStatusFailed)
		return dto.BackupSummary{}, wrapBackupInternal(err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		_ = s.repo.UpdateStatus(ctx, id, BackupStatusFailed)
		return dto.BackupSummary{}, wrapBackupInternal(err)
	}

	sum := sha256.Sum256(content)
	checksum := "sha256:" + hex.EncodeToString(sum[:])
	if err := s.repo.UpdateVerification(ctx, id, BackupStatusSuccess, int64(len(content)), checksum, path); err != nil {
		_ = s.repo.UpdateStatus(ctx, id, BackupStatusFailed)
		return dto.BackupSummary{}, wrapBackupInternal(err)
	}

	backup.Status = BackupStatusSuccess
	backup.SizeBytes = int64(len(content))
	backup.Checksum = checksum
	backup.FilePath = path
	backup.UpdatedAt = time.Now()
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

func (s *backupService) VerifyArtifact(ctx context.Context, id int64) (dto.BackupSummary, error) {
	if id <= 0 {
		return dto.BackupSummary{}, badBackupRequest(errors.New("backup id must be positive"))
	}

	backup, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.BackupSummary{}, wrapBackupInternal(err)
	}
	if backup == nil {
		return dto.BackupSummary{}, apperror.ErrBackupNotFound
	}
	if backup.StorageType != BackupStorageLocal {
		_ = s.repo.UpdateStatus(ctx, id, BackupStatusFailed)
		return dto.BackupSummary{}, badBackupRequest(errors.New("only local backup storage can verify artifacts in P3"))
	}
	if strings.TrimSpace(backup.FilePath) == "" {
		_ = s.repo.UpdateStatus(ctx, id, BackupStatusFailed)
		return dto.BackupSummary{}, badBackupRequest(errors.New("backup file_path is required for artifact verification"))
	}
	if !backupPathWithinLocalDir(s.localDir, backup.FilePath) {
		_ = s.repo.UpdateStatus(ctx, id, BackupStatusFailed)
		return dto.BackupSummary{}, badBackupRequest(errors.New("backup artifact path must be inside BACKUP_LOCAL_DIR"))
	}

	sizeBytes, checksum, err := checksumLocalBackupFile(backup.FilePath)
	if err != nil {
		_ = s.repo.UpdateStatus(ctx, id, BackupStatusFailed)
		return dto.BackupSummary{}, wrapBackupInternal(err)
	}
	if backup.Checksum != "" && backup.Checksum != checksum {
		_ = s.repo.UpdateStatus(ctx, id, BackupStatusFailed)
		return dto.BackupSummary{}, badBackupRequest(errors.New("backup checksum mismatch"))
	}
	if backup.SizeBytes > 0 && backup.SizeBytes != sizeBytes {
		_ = s.repo.UpdateStatus(ctx, id, BackupStatusFailed)
		return dto.BackupSummary{}, badBackupRequest(errors.New("backup size mismatch"))
	}

	if err := s.repo.UpdateVerification(ctx, id, BackupStatusSuccess, sizeBytes, checksum, backup.FilePath); err != nil {
		return dto.BackupSummary{}, wrapBackupInternal(err)
	}

	backup.Status = BackupStatusSuccess
	backup.SizeBytes = sizeBytes
	backup.Checksum = checksum
	backup.UpdatedAt = time.Now()
	return mapBackupSummary(*backup), nil
}

func (s *backupService) MarkStatus(ctx context.Context, id int64, status string) (dto.BackupSummary, error) {
	if id <= 0 {
		return dto.BackupSummary{}, badBackupRequest(errors.New("backup id must be positive"))
	}
	normalized, err := normalizeBackupStatusFilter(status)
	if err != nil || normalized == "" {
		if err == nil {
			err = errors.New("backup status is required")
		}
		return dto.BackupSummary{}, badBackupRequest(err)
	}

	backup, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.BackupSummary{}, wrapBackupInternal(err)
	}
	if backup == nil {
		return dto.BackupSummary{}, apperror.ErrBackupNotFound
	}
	if err := s.repo.UpdateStatus(ctx, id, normalized); err != nil {
		return dto.BackupSummary{}, wrapBackupInternal(err)
	}

	backup.Status = normalized
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

func (s *backupService) CleanupRetention(
	ctx context.Context,
	req dto.BackupRetentionCleanupRequest,
) (dto.BackupRetentionCleanupResult, error) {
	retentionDays := req.RetentionDays
	if retentionDays <= 0 {
		retentionDays = s.retentionDays
	}
	if retentionDays <= 0 {
		retentionDays = 30
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
	matchedRows, err := s.repo.CountDeletableBefore(ctx, cutoff)
	if err != nil {
		return dto.BackupRetentionCleanupResult{}, wrapBackupInternal(err)
	}

	result := dto.BackupRetentionCleanupResult{
		DryRun:              req.DryRun,
		RetentionDays:       retentionDays,
		Cutoff:              cutoff.Format(time.RFC3339),
		MatchedRows:         matchedRows,
		ArchiveBeforeDelete: req.ArchiveBeforeDelete,
	}
	if req.DryRun {
		return result, nil
	}

	deletedRows, err := s.repo.DeleteDeletableBefore(ctx, cutoff)
	if err != nil {
		return dto.BackupRetentionCleanupResult{}, wrapBackupInternal(err)
	}
	result.DeletedRows = deletedRows
	return result, nil
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

func backupArtifactScope(resourceType string) []string {
	switch resourceType {
	case BackupResourcePostgres:
		return []string{"postgres.metadata"}
	case BackupResourceAppMetadata:
		return []string{"application.metadata"}
	case BackupResourceObservability:
		return []string{"observability.config"}
	case BackupResourceCoreAgentTemplate:
		return []string{"core_agent.config"}
	default:
		return []string{"unknown"}
	}
}

func backupArtifactNote(resourceType string) string {
	if resourceType == BackupResourcePostgres {
		return "This P3 artifact is a controlled manifest. Full pg_dump generation remains a follow-up."
	}
	return ""
}

func safeBackupArtifactPath(localDir string, backup model.Backup, now time.Time) (string, error) {
	localDir = strings.TrimSpace(localDir)
	if localDir == "" {
		return "", errors.New("backup local directory is required")
	}
	localAbs, err := filepath.Abs(localDir)
	if err != nil {
		return "", err
	}
	fileName := fmt.Sprintf(
		"snowpanel-%s-%s-%d-%d.json",
		sanitizeBackupFilenamePart(backup.ResourceType),
		sanitizeBackupFilenamePart(backup.ResourceID),
		backup.ID,
		now.Unix(),
	)
	path := filepath.Join(localAbs, fileName)
	rel, err := filepath.Rel(localAbs, path)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", errors.New("backup artifact path escapes local directory")
	}
	return path, nil
}

func backupPathWithinLocalDir(localDir string, path string) bool {
	localAbs, err := filepath.Abs(strings.TrimSpace(localDir))
	if err != nil {
		return false
	}
	pathAbs, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return false
	}
	localReal, err := filepath.EvalSymlinks(localAbs)
	if err == nil {
		localAbs = localReal
	}
	pathReal, err := filepath.EvalSymlinks(pathAbs)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(localAbs, pathReal)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}

func checksumLocalBackupFile(path string) (int64, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer file.Close()

	hash := sha256.New()
	sizeBytes, err := io.Copy(hash, file)
	if err != nil {
		return 0, "", err
	}
	return sizeBytes, "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func sanitizeBackupFilenamePart(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	var builder strings.Builder
	lastDash := false
	for _, char := range value {
		ok := (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')
		if ok {
			builder.WriteRune(char)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "backup"
	}
	if len(result) > 64 {
		result = strings.Trim(result[:64], "-")
	}
	if result == "" {
		return "backup"
	}
	return result
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
