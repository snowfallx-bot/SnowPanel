package repository

import (
	"context"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"gorm.io/gorm"
)

type DatabaseInstanceRepository interface {
	List(ctx context.Context) ([]model.DatabaseInstance, error)
	GetByID(ctx context.Context, id int64) (*model.DatabaseInstance, error)
	GetByName(ctx context.Context, name string) (*model.DatabaseInstance, error)
	Create(ctx context.Context, instance *model.DatabaseInstance) error
	Update(ctx context.Context, instance *model.DatabaseInstance) error
	Delete(ctx context.Context, id int64) error
}

type databaseInstanceRepository struct {
	db *gorm.DB
}

func NewDatabaseInstanceRepository(db *gorm.DB) DatabaseInstanceRepository {
	return &databaseInstanceRepository{db: db}
}

func (r *databaseInstanceRepository) List(ctx context.Context) ([]model.DatabaseInstance, error) {
	var instances []model.DatabaseInstance
	err := r.db.WithContext(ctx).Find(&instances).Error
	if err != nil {
		return nil, err
	}
	return instances, nil
}

func (r *databaseInstanceRepository) GetByID(ctx context.Context, id int64) (*model.DatabaseInstance, error) {
	var instance model.DatabaseInstance
	err := r.db.WithContext(ctx).First(&instance, id).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (r *databaseInstanceRepository) GetByName(ctx context.Context, name string) (*model.DatabaseInstance, error) {
	var instance model.DatabaseInstance
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&instance).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (r *databaseInstanceRepository) Create(ctx context.Context, instance *model.DatabaseInstance) error {
	return r.db.WithContext(ctx).Create(instance).Error
}

func (r *databaseInstanceRepository) Update(ctx context.Context, instance *model.DatabaseInstance) error {
	return r.db.WithContext(ctx).Save(instance).Error
}

func (r *databaseInstanceRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.DatabaseInstance{}, id).Error
}
