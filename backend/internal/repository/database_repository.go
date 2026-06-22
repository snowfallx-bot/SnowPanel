package repository

import (
	"context"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"gorm.io/gorm"
)

type DatabaseRepository interface {
	ListByInstanceID(ctx context.Context, instanceID int64) ([]model.Database, error)
	GetByName(ctx context.Context, instanceID int64, name string) (*model.Database, error)
	Create(ctx context.Context, db *model.Database) error
	Update(ctx context.Context, db *model.Database) error
	Delete(ctx context.Context, id int64) error
	DeleteByInstanceID(ctx context.Context, instanceID int64) error
}

type databaseRepository struct {
	db *gorm.DB
}

func NewDatabaseRepository(db *gorm.DB) DatabaseRepository {
	return &databaseRepository{db: db}
}

func (r *databaseRepository) ListByInstanceID(ctx context.Context, instanceID int64) ([]model.Database, error) {
	var dbs []model.Database
	err := r.db.WithContext(ctx).Where("instance_id = ?", instanceID).Find(&dbs).Error
	if err != nil {
		return nil, err
	}
	return dbs, nil
}

func (r *databaseRepository) GetByName(ctx context.Context, instanceID int64, name string) (*model.Database, error) {
	var db model.Database
	err := r.db.WithContext(ctx).Where("instance_id = ? AND name = ?", instanceID, name).First(&db).Error
	if err != nil {
		return nil, err
	}
	return &db, nil
}

func (r *databaseRepository) Create(ctx context.Context, db *model.Database) error {
	return r.db.WithContext(ctx).Create(db).Error
}

func (r *databaseRepository) Update(ctx context.Context, db *model.Database) error {
	return r.db.WithContext(ctx).Save(db).Error
}

func (r *databaseRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Database{}, id).Error
}

func (r *databaseRepository) DeleteByInstanceID(ctx context.Context, instanceID int64) error {
	return r.db.WithContext(ctx).Delete(&model.Database{}, "instance_id = ?", instanceID).Error
}
