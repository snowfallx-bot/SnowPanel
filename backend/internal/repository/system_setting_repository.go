package repository

import (
	"context"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SystemSettingRepository interface {
	GetByKey(ctx context.Context, key string) (*model.SystemSetting, error)
	Upsert(ctx context.Context, setting *model.SystemSetting) error
}

type systemSettingRepository struct {
	db *gorm.DB
}

func NewSystemSettingRepository(db *gorm.DB) SystemSettingRepository {
	return &systemSettingRepository{db: db}
}

func (r *systemSettingRepository) GetByKey(
	ctx context.Context,
	key string,
) (*model.SystemSetting, error) {
	var setting model.SystemSetting
	if err := r.db.WithContext(ctx).First(&setting, "key = ?", key).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &setting, nil
}

func (r *systemSettingRepository) Upsert(
	ctx context.Context,
	setting *model.SystemSetting,
) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"value",
			"value_type",
			"is_encrypted",
			"description",
			"updated_by",
			"updated_at",
		}),
	}).Create(setting).Error
}
