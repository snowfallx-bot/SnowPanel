package repository

import (
	"context"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"gorm.io/gorm"
)

type AuditListFilter struct {
	Page   int
	Size   int
	Module string
	Action string
}

type AuditRepository interface {
	Create(ctx context.Context, item *model.AuditLog) error
	List(ctx context.Context, filter AuditListFilter) ([]model.AuditLog, int64, error)
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

	query := r.db.WithContext(ctx).Model(&model.AuditLog{})
	if filter.Module != "" {
		query = query.Where("module = ?", filter.Module)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}

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
