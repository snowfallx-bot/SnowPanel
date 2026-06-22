package service

import (
	"context"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/repository"
)

type WebsiteDomainService interface {
	ListDomains(ctx context.Context, websiteID int64) ([]dto.WebsiteDomain, error)
	CreateDomain(ctx context.Context, req dto.CreateWebsiteDomainRequest) (dto.WebsiteDomain, error)
	UpdateDomain(ctx context.Context, domainID int64, req dto.UpdateWebsiteDomainRequest) (dto.WebsiteDomain, error)
	DeleteDomain(ctx context.Context, domainID int64) error
}

type websiteDomainService struct {
	domainRepo repository.WebsiteDomainRepository
	websiteRepo repository.WebsiteRepository
	auditService AuditService
}

func NewWebsiteDomainService(
	domainRepo repository.WebsiteDomainRepository,
	websiteRepo repository.WebsiteRepository,
	auditService AuditService,
) WebsiteDomainService {
	return &websiteDomainService{
		domainRepo:    domainRepo,
		websiteRepo:   websiteRepo,
		auditService:  auditService,
	}
}

func (s *websiteDomainService) ListDomains(ctx context.Context, websiteID int64) ([]dto.WebsiteDomain, error) {
	// Check website exists
	website, err := s.websiteRepo.GetByID(ctx, websiteID)
	if err != nil {
		return nil, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if website == nil {
		return nil, apperror.ErrWebsiteNotFound
	}

	domains, err := s.domainRepo.ListByWebsiteID(ctx, websiteID)
	if err != nil {
		return nil, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	result := make([]dto.WebsiteDomain, 0, len(domains))
	for _, d := range domains {
		result = append(result, dto.WebsiteDomain{
			ID:        d.ID,
			WebsiteID: d.WebsiteID,
			Domain:    d.Domain,
			IsPrimary: d.IsPrimary,
			CreatedAt: d.CreatedAt,
		})
	}

	return result, nil
}

func (s *websiteDomainService) CreateDomain(ctx context.Context, req dto.CreateWebsiteDomainRequest) (dto.WebsiteDomain, error) {
	// Check website exists
	website, err := s.websiteRepo.GetByID(ctx, req.WebsiteID)
	if err != nil {
		return dto.WebsiteDomain{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if website == nil {
		return dto.WebsiteDomain{}, apperror.ErrWebsiteNotFound
	}

	// Check if domain already exists
	existing, err := s.domainRepo.GetByDomain(ctx, req.Domain)
	if err != nil {
		return dto.WebsiteDomain{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if existing != nil {
		return dto.WebsiteDomain{}, apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			"domain already in use",
			err,
		)
	}

	domain := &model.WebsiteDomain{
		WebsiteID: req.WebsiteID,
		Domain:    req.Domain,
		IsPrimary: req.IsPrimary,
	}

	if err := s.domainRepo.Create(ctx, domain); err != nil {
		return dto.WebsiteDomain{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	return dto.WebsiteDomain{
		ID:        domain.ID,
		WebsiteID: domain.WebsiteID,
		Domain:    domain.Domain,
		IsPrimary: domain.IsPrimary,
		CreatedAt: domain.CreatedAt,
	}, nil
}

func (s *websiteDomainService) UpdateDomain(ctx context.Context, domainID int64, req dto.UpdateWebsiteDomainRequest) (dto.WebsiteDomain, error) {
	domain, err := s.domainRepo.GetByID(ctx, domainID)
	if err != nil {
		return dto.WebsiteDomain{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if domain == nil {
		return dto.WebsiteDomain{}, apperror.ErrWebsiteDomainNotFound
	}

	if req.Domain != nil {
		// Check if new domain conflicts with another record
		if *req.Domain != domain.Domain {
			existing, err := s.domainRepo.GetByDomain(ctx, *req.Domain)
			if err != nil {
				return dto.WebsiteDomain{}, apperror.Wrap(
					apperror.ErrInternal.Code,
					apperror.ErrInternal.HTTPStatus,
					apperror.ErrInternal.Message,
					err,
				)
			}
			if existing != nil && existing.ID != domainID {
				return dto.WebsiteDomain{}, apperror.Wrap(
					apperror.ErrBadRequest.Code,
					apperror.ErrBadRequest.HTTPStatus,
					"domain already in use",
					err,
				)
			}
			domain.Domain = *req.Domain
		}
	}

	if req.IsPrimary != nil {
		domain.IsPrimary = *req.IsPrimary
	}

	if err := s.domainRepo.Update(ctx, domain); err != nil {
		return dto.WebsiteDomain{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	return dto.WebsiteDomain{
		ID:        domain.ID,
		WebsiteID: domain.WebsiteID,
		Domain:    domain.Domain,
		IsPrimary: domain.IsPrimary,
		CreatedAt: domain.CreatedAt,
	}, nil
}

func (s *websiteDomainService) DeleteDomain(ctx context.Context, domainID int64) error {
	domain, err := s.domainRepo.GetByID(ctx, domainID)
	if err != nil {
		return apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if domain == nil {
		return apperror.ErrWebsiteDomainNotFound
	}

	if err := s.domainRepo.Delete(ctx, domainID); err != nil {
		return apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	return nil
}
