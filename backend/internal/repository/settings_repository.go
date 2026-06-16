package repository

import (
	"context"
	"errors"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"gorm.io/gorm"
)

type SettingsRepository interface {
	List(ctx context.Context) ([]model.SystemSetting, error)
	GetByKey(ctx context.Context, key string) (*model.SystemSetting, error)
	Create(ctx context.Context, setting *model.SystemSetting) error
	Update(ctx context.Context, setting *model.SystemSetting) error
	Delete(ctx context.Context, key string) error
}

type settingsRepository struct {
	db *gorm.DB
}

func NewSettingsRepository(db *gorm.DB) SettingsRepository {
	return &settingsRepository{db: db}
}

func (r *settingsRepository) List(ctx context.Context) ([]model.SystemSetting, error) {
	var settings []model.SystemSetting
	err := r.db.WithContext(ctx).Find(&settings).Error
	if err != nil {
		return nil, err
	}
	return settings, nil
}

func (r *settingsRepository) GetByKey(ctx context.Context, key string) (*model.SystemSetting, error) {
	var setting model.SystemSetting
	err := r.db.WithContext(ctx).Where("key = ?", key).First(&setting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

func (r *settingsRepository) Create(ctx context.Context, setting *model.SystemSetting) error {
	return r.db.WithContext(ctx).Create(setting).Error
}

func (r *settingsRepository) Update(ctx context.Context, setting *model.SystemSetting) error {
	return r.db.WithContext(ctx).Save(setting).Error
}

func (r *settingsRepository) Delete(ctx context.Context, key string) error {
	return r.db.WithContext(ctx).Delete(&model.SystemSetting{}, "key = ?", key).Error
}
