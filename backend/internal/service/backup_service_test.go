package service

import (
	"context"
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
