package repository

import (
	"context"
	"strings"
	"time"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"gorm.io/gorm"
)

type BackupListFilter struct {
	Page         int
	Size         int
	Status       string
	ResourceType string
	ResourceID   string
}

type BackupRepository interface {
	Create(ctx context.Context, backup *model.Backup) error
	GetByID(ctx context.Context, id int64) (*model.Backup, error)
	List(ctx context.Context, filter BackupListFilter) ([]model.Backup, int64, error)
	UpdateVerification(ctx context.Context, id int64, status string, sizeBytes int64, checksum string, filePath string) error
	UpdateStatus(ctx context.Context, id int64, status string) error
	CountDeletableBefore(ctx context.Context, cutoff time.Time) (int64, error)
	ListDeletableBefore(ctx context.Context, cutoff time.Time) ([]model.Backup, error)
	DeleteDeletableBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type backupRepository struct {
	db *gorm.DB
}

func NewBackupRepository(db *gorm.DB) BackupRepository {
	return &backupRepository{db: db}
}

func (r *backupRepository) Create(ctx context.Context, backup *model.Backup) error {
	return r.db.WithContext(ctx).Create(backup).Error
}

func (r *backupRepository) GetByID(ctx context.Context, id int64) (*model.Backup, error) {
	var backup model.Backup
	if err := r.db.WithContext(ctx).First(&backup, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &backup, nil
}

func (r *backupRepository) List(
	ctx context.Context,
	filter BackupListFilter,
) ([]model.Backup, int64, error) {
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	size := filter.Size
	if size <= 0 {
		size = 20
	}
	if size > 100 {
		size = 100
	}

	query := r.db.WithContext(ctx).Model(&model.Backup{})
	if status := strings.TrimSpace(filter.Status); status != "" {
		query = query.Where("status = ?", status)
	}
	if resourceType := strings.TrimSpace(filter.ResourceType); resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if resourceID := strings.TrimSpace(filter.ResourceID); resourceID != "" {
		query = query.Where("resource_id = ?", resourceID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []model.Backup
	if err := query.
		Order("created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *backupRepository) UpdateVerification(
	ctx context.Context,
	id int64,
	status string,
	sizeBytes int64,
	checksum string,
	filePath string,
) error {
	updates := map[string]interface{}{
		"status":     status,
		"size_bytes": sizeBytes,
		"checksum":   checksum,
	}
	if strings.TrimSpace(filePath) != "" {
		updates["file_path"] = filePath
	}
	return r.db.WithContext(ctx).Model(&model.Backup{}).Where("id = ?", id).Updates(updates).Error
}

func (r *backupRepository) UpdateStatus(ctx context.Context, id int64, status string) error {
	return r.db.WithContext(ctx).Model(&model.Backup{}).Where("id = ?", id).Update("status", status).Error
}

func (r *backupRepository) CountDeletableBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&model.Backup{}).
		Where("created_at < ? AND status IN ?", cutoff, []string{"success", "failed"}).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *backupRepository) ListDeletableBefore(ctx context.Context, cutoff time.Time) ([]model.Backup, error) {
	var items []model.Backup
	if err := r.db.WithContext(ctx).
		Where("created_at < ? AND status IN ?", cutoff, []string{"success", "failed"}).
		Order("created_at ASC, id ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *backupRepository) DeleteDeletableBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("created_at < ? AND status IN ?", cutoff, []string{"success", "failed"}).
		Delete(&model.Backup{})
	return result.RowsAffected, result.Error
}
