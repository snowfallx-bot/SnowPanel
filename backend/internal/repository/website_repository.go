package repository

import (
	"context"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"gorm.io/gorm"
)

type WebsiteRepository interface {
	List(ctx context.Context) ([]model.Website, error)
	GetByID(ctx context.Context, id int64) (*model.Website, error)
	GetByName(ctx context.Context, name string) (*model.Website, error)
	Create(ctx context.Context, website *model.Website) error
	Update(ctx context.Context, website *model.Website) error
	Delete(ctx context.Context, id int64) error
}

type websiteRepository struct {
	db *gorm.DB
}

func NewWebsiteRepository(db *gorm.DB) WebsiteRepository {
	return &websiteRepository{db: db}
}

func (r *websiteRepository) List(ctx context.Context) ([]model.Website, error) {
	var websites []model.Website
	err := r.db.WithContext(ctx).Find(&websites).Error
	if err != nil {
		return nil, err
	}
	return websites, nil
}

func (r *websiteRepository) GetByID(ctx context.Context, id int64) (*model.Website, error) {
	var website model.Website
	err := r.db.WithContext(ctx).First(&website, id).Error
	if err != nil {
		return nil, err
	}
	return &website, nil
}

func (r *websiteRepository) GetByName(ctx context.Context, name string) (*model.Website, error) {
	var website model.Website
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&website).Error
	if err != nil {
		return nil, err
	}
	return &website, nil
}

func (r *websiteRepository) Create(ctx context.Context, website *model.Website) error {
	return r.db.WithContext(ctx).Create(website).Error
}

func (r *websiteRepository) Update(ctx context.Context, website *model.Website) error {
	return r.db.WithContext(ctx).Save(website).Error
}

func (r *websiteRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Website{}, id).Error
}
