package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/repository"
)

const (
	testChecksumA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	testChecksumB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func TestBackupMetadataCreateDefaults(t *testing.T) {
	repo := newFakeBackupRepo()
	service := NewBackupService(repo)
	createdBy := int64(7)

	result, err := service.CreateMetadata(context.Background(), dto.CreateBackupMetadataRequest{
		ResourceType: "postgres",
		ResourceID:   "primary",
		FilePath:     "backups/postgres.dump.sql",
	}, &createdBy)
	if err != nil {
		t.Fatalf("CreateMetadata returned error: %v", err)
	}

	if result.ID == 0 {
		t.Fatalf("expected backup id to be assigned")
	}
	if result.Status != BackupStatusPending {
		t.Fatalf("expected pending status, got %q", result.Status)
	}
	if result.StorageType != BackupStorageLocal {
		t.Fatalf("expected local storage, got %q", result.StorageType)
	}
	if result.CreatedBy == nil || *result.CreatedBy != createdBy {
		t.Fatalf("expected created_by to be preserved")
	}
}

func TestBackupCreateArtifactWritesLocalManifest(t *testing.T) {
	repo := newFakeBackupRepo()
	localDir := t.TempDir()
	service := NewBackupServiceWithOptions(repo, BackupServiceOptions{LocalDir: localDir})
	result, err := service.CreateMetadata(context.Background(), dto.CreateBackupMetadataRequest{
		ResourceType: BackupResourceAppMetadata,
		ResourceID:   "primary/app",
		StorageType:  BackupStorageLocal,
	}, nil)
	if err != nil {
		t.Fatalf("CreateMetadata returned error: %v", err)
	}

	artifact, err := service.CreateArtifact(context.Background(), result.ID)
	if err != nil {
		t.Fatalf("CreateArtifact returned error: %v", err)
	}

	if artifact.Status != BackupStatusSuccess {
		t.Fatalf("expected success status, got %q", artifact.Status)
	}
	if artifact.SizeBytes <= 0 {
		t.Fatalf("expected size to be recorded, got %d", artifact.SizeBytes)
	}
	if !strings.HasPrefix(artifact.Checksum, "sha256:") || len(artifact.Checksum) != len("sha256:")+64 {
		t.Fatalf("expected sha256 checksum, got %q", artifact.Checksum)
	}
	rel, err := filepath.Rel(localDir, artifact.FilePath)
	if err != nil {
		t.Fatalf("failed to compare artifact path: %v", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		t.Fatalf("artifact path escaped local dir: %s", artifact.FilePath)
	}

	content, err := os.ReadFile(artifact.FilePath)
	if err != nil {
		t.Fatalf("failed to read artifact: %v", err)
	}
	if int64(len(content)) != artifact.SizeBytes {
		t.Fatalf("expected artifact size %d, got %d", artifact.SizeBytes, len(content))
	}
	if strings.Contains(strings.ToLower(string(content)), "secret") ||
		strings.Contains(strings.ToLower(string(content)), "password") {
		t.Fatalf("artifact manifest should not contain secret-looking fields: %s", string(content))
	}
	var manifest backupArtifactManifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		t.Fatalf("failed to parse artifact manifest: %v", err)
	}
	if manifest.BackupID != result.ID || manifest.ResourceType != BackupResourceAppMetadata {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}

	stored := repo.items[result.ID]
	if stored.FilePath != artifact.FilePath || stored.Checksum != artifact.Checksum || stored.SizeBytes != artifact.SizeBytes {
		t.Fatalf("repository verification was not updated: stored=%+v artifact=%+v", stored, artifact)
	}
}

func TestBackupMetadataCreateRejectsOutOfScopeResource(t *testing.T) {
	service := NewBackupService(newFakeBackupRepo())

	_, err := service.CreateMetadata(context.Background(), dto.CreateBackupMetadataRequest{
		ResourceType: "filesystem",
		ResourceID:   "/etc",
	}, nil)
	if err == nil {
		t.Fatalf("expected out-of-scope resource to fail")
	}
	appErr, ok := apperror.As(err)
	if !ok || appErr.Code != apperror.ErrBadRequest.Code {
		t.Fatalf("expected bad request, got %v", err)
	}
}

func TestBackupVerifyUpdatesChecksumAndStatus(t *testing.T) {
	repo := newFakeBackupRepo()
	service := NewBackupService(repo)
	result, err := service.CreateMetadata(context.Background(), dto.CreateBackupMetadataRequest{
		ResourceType: BackupResourcePostgres,
		ResourceID:   "primary",
	}, nil)
	if err != nil {
		t.Fatalf("CreateMetadata returned error: %v", err)
	}

	verified, err := service.Verify(context.Background(), result.ID, dto.VerifyBackupRequest{
		SizeBytes: 2048,
		Checksum:  testChecksumA,
		FilePath:  "backups/postgres.dump.sql",
	})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}

	if verified.Status != BackupStatusSuccess {
		t.Fatalf("expected success status, got %q", verified.Status)
	}
	if verified.Checksum != "sha256:"+testChecksumA {
		t.Fatalf("expected normalized checksum, got %q", verified.Checksum)
	}
	if verified.SizeBytes != 2048 {
		t.Fatalf("expected size 2048, got %d", verified.SizeBytes)
	}
}

func TestBackupVerifyMismatchMarksFailed(t *testing.T) {
	repo := newFakeBackupRepo()
	now := time.Now()
	repo.items[1] = model.Backup{
		ID:           1,
		ResourceType: BackupResourcePostgres,
		ResourceID:   "primary",
		StorageType:  BackupStorageLocal,
		FilePath:     "backups/postgres.dump.sql",
		SizeBytes:    2048,
		Checksum:     "sha256:" + testChecksumA,
		Status:       BackupStatusSuccess,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	_, err := NewBackupService(repo).Verify(context.Background(), 1, dto.VerifyBackupRequest{
		SizeBytes: 2048,
		Checksum:  testChecksumB,
	})
	if err == nil {
		t.Fatalf("expected checksum mismatch to fail")
	}
	appErr, ok := apperror.As(err)
	if !ok || appErr.Code != apperror.ErrBadRequest.Code {
		t.Fatalf("expected bad request, got %v", err)
	}
	if repo.items[1].Status != BackupStatusFailed {
		t.Fatalf("expected mismatch to mark backup failed, got %q", repo.items[1].Status)
	}
}

func TestBackupMarkStatusUpdatesExistingBackup(t *testing.T) {
	repo := newFakeBackupRepo()
	service := NewBackupService(repo)
	result, err := service.CreateMetadata(context.Background(), dto.CreateBackupMetadataRequest{
		ResourceType: BackupResourcePostgres,
		ResourceID:   "primary",
	}, nil)
	if err != nil {
		t.Fatalf("CreateMetadata returned error: %v", err)
	}

	updated, err := service.MarkStatus(context.Background(), result.ID, BackupStatusRunning)
	if err != nil {
		t.Fatalf("MarkStatus returned error: %v", err)
	}
	if updated.Status != BackupStatusRunning {
		t.Fatalf("expected running status, got %q", updated.Status)
	}
}

func TestBackupListNormalizesFilters(t *testing.T) {
	repo := newFakeBackupRepo()
	now := time.Now()
	repo.items[1] = model.Backup{
		ID:           1,
		ResourceType: BackupResourcePostgres,
		ResourceID:   "primary",
		StorageType:  BackupStorageLocal,
		Status:       BackupStatusSuccess,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	result, err := NewBackupService(repo).List(context.Background(), dto.ListBackupsQuery{
		Page:         1,
		Size:         10,
		Status:       strings.ToUpper(BackupStatusSuccess),
		ResourceType: strings.ToUpper(BackupResourcePostgres),
		ResourceID:   "primary",
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("expected one backup, got total=%d len=%d", result.Total, len(result.Items))
	}
}

func TestBackupRetentionCleanupDryRunDoesNotDeletePendingOrRunning(t *testing.T) {
	repo := newFakeBackupRepo()
	old := time.Now().AddDate(0, 0, -60)
	repo.items[1] = model.Backup{ID: 1, Status: BackupStatusSuccess, CreatedAt: old, UpdatedAt: old}
	repo.items[2] = model.Backup{ID: 2, Status: BackupStatusFailed, CreatedAt: old, UpdatedAt: old}
	repo.items[3] = model.Backup{ID: 3, Status: BackupStatusPending, CreatedAt: old, UpdatedAt: old}
	repo.items[4] = model.Backup{ID: 4, Status: BackupStatusRunning, CreatedAt: old, UpdatedAt: old}
	service := NewBackupServiceWithOptions(repo, BackupServiceOptions{RetentionDays: 30})

	result, err := service.CleanupRetention(context.Background(), dto.BackupRetentionCleanupRequest{
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("CleanupRetention returned error: %v", err)
	}
	if !result.DryRun || result.MatchedRows != 2 || result.DeletedRows != 0 {
		t.Fatalf("unexpected dry run result: %+v", result)
	}
	if len(repo.items) != 4 {
		t.Fatalf("dry run deleted rows unexpectedly")
	}
}

func TestBackupRetentionCleanupDeletesTerminalRowsOnly(t *testing.T) {
	repo := newFakeBackupRepo()
	old := time.Now().AddDate(0, 0, -60)
	recent := time.Now().AddDate(0, 0, -5)
	repo.items[1] = model.Backup{ID: 1, Status: BackupStatusSuccess, CreatedAt: old, UpdatedAt: old}
	repo.items[2] = model.Backup{ID: 2, Status: BackupStatusFailed, CreatedAt: old, UpdatedAt: old}
	repo.items[3] = model.Backup{ID: 3, Status: BackupStatusPending, CreatedAt: old, UpdatedAt: old}
	repo.items[4] = model.Backup{ID: 4, Status: BackupStatusSuccess, CreatedAt: recent, UpdatedAt: recent}
	service := NewBackupServiceWithOptions(repo, BackupServiceOptions{RetentionDays: 30})

	result, err := service.CleanupRetention(context.Background(), dto.BackupRetentionCleanupRequest{})
	if err != nil {
		t.Fatalf("CleanupRetention returned error: %v", err)
	}
	if result.DryRun || result.MatchedRows != 2 || result.DeletedRows != 2 {
		t.Fatalf("unexpected cleanup result: %+v", result)
	}
	if _, ok := repo.items[1]; ok {
		t.Fatalf("expected old success backup to be deleted")
	}
	if _, ok := repo.items[2]; ok {
		t.Fatalf("expected old failed backup to be deleted")
	}
	if _, ok := repo.items[3]; !ok {
		t.Fatalf("pending backup should not be deleted")
	}
	if _, ok := repo.items[4]; !ok {
		t.Fatalf("recent backup should not be deleted")
	}
}

type fakeBackupRepo struct {
	nextID int64
	items  map[int64]model.Backup
}

func newFakeBackupRepo() *fakeBackupRepo {
	return &fakeBackupRepo{
		nextID: 1,
		items:  map[int64]model.Backup{},
	}
}

func (r *fakeBackupRepo) Create(_ context.Context, backup *model.Backup) error {
	backup.ID = r.nextID
	r.nextID++
	now := time.Now()
	backup.CreatedAt = now
	backup.UpdatedAt = now
	r.items[backup.ID] = *backup
	return nil
}

func (r *fakeBackupRepo) GetByID(_ context.Context, id int64) (*model.Backup, error) {
	backup, ok := r.items[id]
	if !ok {
		return nil, nil
	}
	return &backup, nil
}

func (r *fakeBackupRepo) List(
	_ context.Context,
	filter repository.BackupListFilter,
) ([]model.Backup, int64, error) {
	var items []model.Backup
	for _, item := range r.items {
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		if filter.ResourceType != "" && item.ResourceType != filter.ResourceType {
			continue
		}
		if filter.ResourceID != "" && item.ResourceID != filter.ResourceID {
			continue
		}
		items = append(items, item)
	}
	return items, int64(len(items)), nil
}

func (r *fakeBackupRepo) UpdateVerification(
	_ context.Context,
	id int64,
	status string,
	sizeBytes int64,
	checksum string,
	filePath string,
) error {
	item := r.items[id]
	item.Status = status
	item.SizeBytes = sizeBytes
	item.Checksum = checksum
	if filePath != "" {
		item.FilePath = filePath
	}
	item.UpdatedAt = time.Now()
	r.items[id] = item
	return nil
}

func (r *fakeBackupRepo) UpdateStatus(_ context.Context, id int64, status string) error {
	item := r.items[id]
	item.Status = status
	item.UpdatedAt = time.Now()
	r.items[id] = item
	return nil
}

func (r *fakeBackupRepo) CountDeletableBefore(_ context.Context, cutoff time.Time) (int64, error) {
	var count int64
	for _, item := range r.items {
		if backupIsDeletableBefore(item, cutoff) {
			count++
		}
	}
	return count, nil
}

func (r *fakeBackupRepo) DeleteDeletableBefore(_ context.Context, cutoff time.Time) (int64, error) {
	var deleted int64
	for id, item := range r.items {
		if backupIsDeletableBefore(item, cutoff) {
			delete(r.items, id)
			deleted++
		}
	}
	return deleted, nil
}

func backupIsDeletableBefore(item model.Backup, cutoff time.Time) bool {
	if !item.CreatedAt.Before(cutoff) {
		return false
	}
	return item.Status == BackupStatusSuccess || item.Status == BackupStatusFailed
}
