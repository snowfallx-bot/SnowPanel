package repository

import (
	"context"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"gorm.io/gorm"
)

type WebsiteDomainRepository interface {
	ListByWebsiteID(ctx context.Context, websiteID int64) ([]model.WebsiteDomain, error)
	GetByID(ctx context.Context, id int64) (*model.WebsiteDomain, error)
	GetByDomain(ctx context.Context, domain string) (*model.WebsiteDomain, error)
	Create(ctx context.Context, domain *model.WebsiteDomain) error
	Update(ctx context.Context, domain *model.WebsiteDomain) error
	Delete(ctx context.Context, id int64) error
	DeleteByWebsiteID(ctx context.Context, websiteID int64) error
}

type websiteDomainRepository struct {
	db *gorm.DB
}

func NewWebsiteDomainRepository(db *gorm.DB) WebsiteDomainRepository {
	return &websiteDomainRepository{db: db}
}

func (r *websiteDomainRepository) ListByWebsiteID(ctx context.Context, websiteID int64) ([]model.WebsiteDomain, error) {
	var domains []model.WebsiteDomain
	err := r.db.WithContext(ctx).Where("website_id = ?", websiteID).Find(&domains).Error
	if err != nil {
		return nil, err
	}
	return domains, nil
}

func (r *websiteDomainRepository) GetByDomain(ctx context.Context, domain string) (*model.WebsiteDomain, error) {
	var domainModel model.WebsiteDomain
	err := r.db.WithContext(ctx).Where("domain = ?", domain).First(&domainModel).Error
	if err != nil {
		return nil, err
	}
	return &domainModel, nil
}

func (r *websiteDomainRepository) GetByID(ctx context.Context, id int64) (*model.WebsiteDomain, error) {
	var domainModel model.WebsiteDomain
	err := r.db.WithContext(ctx).First(&domainModel, id).Error
	if err != nil {
		return nil, err
	}
	return &domainModel, nil
}

func (r *websiteDomainRepository) Create(ctx context.Context, domain *model.WebsiteDomain) error {
	return r.db.WithContext(ctx).Create(domain).Error
}

func (r *websiteDomainRepository) Update(ctx context.Context, domain *model.WebsiteDomain) error {
	return r.db.WithContext(ctx).Save(domain).Error
}

func (r *websiteDomainRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.WebsiteDomain{}, id).Error
}

func (r *websiteDomainRepository) DeleteByWebsiteID(ctx context.Context, websiteID int64) error {
	return r.db.WithContext(ctx).Delete(&model.WebsiteDomain{}, "website_id = ?", websiteID).Error
}
