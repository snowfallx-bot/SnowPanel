package repository

import (
	"context"
	"time"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"gorm.io/gorm"
)

type AuditListFilter struct {
	Page       int
	Size       int
	StartTime  *time.Time
	EndTime    *time.Time
	UserID     *int64
	Username   string
	Module     string
	Action     string
	TargetType string
	TargetID   string
	Success    *bool
	ResultCode string
	RequestID  string
	TraceID    string
}

type AuditRepository interface {
	Create(ctx context.Context, item *model.AuditLog) error
	List(ctx context.Context, filter AuditListFilter) ([]model.AuditLog, int64, error)
	CountBefore(ctx context.Context, cutoff time.Time) (int64, error)
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type auditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) AuditRepository {
	return &auditRepository{db: db}
}

func (r *auditRepository) Create(ctx context.Context, item *model.AuditLog) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *auditRepository) List(
	ctx context.Context,
	filter AuditListFilter,
) ([]model.AuditLog, int64, error) {
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

	query := applyAuditListFilter(r.db.WithContext(ctx).Model(&model.AuditLog{}), filter)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []model.AuditLog
	if err := query.
		Order("created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *auditRepository) CountBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.AuditLog{}).
		Where("created_at < ?", cutoff).
		Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (r *auditRepository) DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("created_at < ?", cutoff).
		Delete(&model.AuditLog{})
	return result.RowsAffected, result.Error
}

func applyAuditListFilter(query *gorm.DB, filter AuditListFilter) *gorm.DB {
	if filter.StartTime != nil {
		query = query.Where("created_at >= ?", *filter.StartTime)
	}
	if filter.EndTime != nil {
		query = query.Where("created_at <= ?", *filter.EndTime)
	}
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.Username != "" {
		query = query.Where("username = ?", filter.Username)
	}
	if filter.Module != "" {
		query = query.Where("module = ?", filter.Module)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.TargetType != "" {
		query = query.Where("target_type = ?", filter.TargetType)
	}
	if filter.TargetID != "" {
		query = query.Where("target_id = ?", filter.TargetID)
	}
	if filter.Success != nil {
		query = query.Where("success = ?", *filter.Success)
	}
	if filter.ResultCode != "" {
		query = query.Where("result_code = ?", filter.ResultCode)
	}
	if filter.RequestID != "" {
		query = query.Where("request_id = ?", filter.RequestID)
	}
	if filter.TraceID != "" {
		query = query.Where("trace_id = ?", filter.TraceID)
	}
	return query
}
