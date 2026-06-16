package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/grpcclient"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/repository"
)

type WebsiteService interface {
	ListWebsites(ctx context.Context) ([]dto.WebsiteListItem, error)
	GetWebsite(ctx context.Context, id int64) (dto.Website, error)
	CreateWebsite(ctx context.Context, req dto.CreateWebsiteRequest) (dto.Website, error)
	UpdateWebsite(ctx context.Context, id int64, req dto.UpdateWebsiteRequest) (dto.Website, error)
	DeleteWebsite(ctx context.Context, id int64) error
	EnableWebsite(ctx context.Context, id int64) error
	DisableWebsite(ctx context.Context, id int64) error
}

type websiteService struct {
	websiteRepo        repository.WebsiteRepository
	websiteDomainRepo  repository.WebsiteDomainRepository
	auditService       AuditService
	agentClient        grpcclient.AgentClient
}

func NewWebsiteService(
	websiteRepo repository.WebsiteRepository,
	websiteDomainRepo repository.WebsiteDomainRepository,
	auditService AuditService,
	agentClient grpcclient.AgentClient,
) WebsiteService {
	return &websiteService{
		websiteRepo:        websiteRepo,
		websiteDomainRepo:  websiteDomainRepo,
		auditService:       auditService,
		agentClient:        agentClient,
	}
}

func (s *websiteService) ListWebsites(ctx context.Context) ([]dto.WebsiteListItem, error) {
	websites, err := s.websiteRepo.List(ctx)
	if err != nil {
		return nil, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	result := make([]dto.WebsiteListItem, 0, len(websites))
	for _, w := range websites {
		result = append(result, dto.WebsiteListItem{
			ID:        w.ID,
			Name:      w.Name,
			RootPath:  w.RootPath,
			Runtime:   w.Runtime,
			Status:    w.Status,
			HostID:    w.HostID,
			CreatedAt: w.CreatedAt,
			UpdatedAt: w.UpdatedAt,
		})
	}

	return result, nil
}

func (s *websiteService) GetWebsite(ctx context.Context, id int64) (dto.Website, error) {
	website, err := s.websiteRepo.GetByID(ctx, id)
	if err != nil {
		return dto.Website{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if website == nil {
		return dto.Website{}, apperror.ErrWebsiteNotFound
	}

	domains, err := s.websiteDomainRepo.ListByWebsiteID(ctx, website.ID)
	if err != nil {
		return dto.Website{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	domainList := make([]string, 0, len(domains))
	for _, d := range domains {
		domainList = append(domainList, d.Domain)
	}

	return dto.Website{
		ID:        website.ID,
		Name:      website.Name,
		RootPath:  website.RootPath,
		Runtime:   website.Runtime,
		Status:    website.Status,
		HostID:    website.HostID,
		Domains:   domainList,
		CreatedAt: website.CreatedAt,
		UpdatedAt: website.UpdatedAt,
	}, nil
}

func (s *websiteService) CreateWebsite(ctx context.Context, req dto.CreateWebsiteRequest) (dto.Website, error) {
	// Validate runtime
	if !isValidRuntime(req.Runtime) {
		return dto.Website{}, apperror.ErrInvalidRuntime
	}

	// Validate domains
	if len(req.Domains) == 0 {
		return dto.Website{}, apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			"at least one domain is required",
			fmt.Errorf("no domains provided"),
		)
	}

	for _, domain := range req.Domains {
		if strings.TrimSpace(domain) == "" {
			return dto.Website{}, apperror.ErrInvalidDomain
		}
	}

	// Check if website name already exists
	existing, err := s.websiteRepo.GetByName(ctx, req.Name)
	if err != nil {
		return dto.Website{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if existing != nil {
		return dto.Website{}, apperror.ErrWebsiteNameExists
	}

	// Create website
	website := &model.Website{
		Name:      req.Name,
		RootPath:  req.RootPath,
		Runtime:   req.Runtime,
		Status:    1, // enabled by default
		HostID:    req.HostID,
	}

	if err := s.websiteRepo.Create(ctx, website); err != nil {
		return dto.Website{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	// Create domains
	for _, domain := range req.Domains {
		domainModel := &model.WebsiteDomain{
			WebsiteID: website.ID,
			Domain:    domain,
			IsPrimary: false, // TODO: set primary based on first domain or explicit flag
		}
		if err := s.websiteDomainRepo.Create(ctx, domainModel); err != nil {
			// Rollback website creation
			_ = s.websiteRepo.Delete(ctx, website.ID)
			return dto.Website{}, apperror.Wrap(
				apperror.ErrInternal.Code,
				apperror.ErrInternal.HTTPStatus,
				apperror.ErrInternal.Message,
				err,
			)
		}
	}

	// Map to DTO
	domainList := make([]string, 0, len(req.Domains))
	for _, d := range req.Domains {
		domainList = append(domainList, d)
	}

	return dto.Website{
		ID:        website.ID,
		Name:      website.Name,
		RootPath:  website.RootPath,
		Runtime:   website.Runtime,
		Status:    website.Status,
		HostID:    website.HostID,
		Domains:   domainList,
		CreatedAt: website.CreatedAt,
		UpdatedAt: website.UpdatedAt,
	}, nil
}

func (s *websiteService) UpdateWebsite(ctx context.Context, id int64, req dto.UpdateWebsiteRequest) (dto.Website, error) {
	website, err := s.websiteRepo.GetByID(ctx, id)
	if err != nil {
		return dto.Website{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if website == nil {
		return dto.Website{}, apperror.ErrWebsiteNotFound
	}

	// Update fields if provided
	if req.RootPath != nil {
		website.RootPath = *req.RootPath
	}
	if req.Runtime != nil {
		if !isValidRuntime(*req.Runtime) {
			return dto.Website{}, apperror.ErrInvalidRuntime
		}
		website.Runtime = *req.Runtime
	}
	if req.Status != nil {
		website.Status = *req.Status
	}
	if req.HostID != nil {
		website.HostID = req.HostID
	}

	if err := s.websiteRepo.Update(ctx, website); err != nil {
		return dto.Website{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	// Update domains if provided
	if req.Domains != nil {
		// Delete existing domains
		if err := s.websiteDomainRepo.DeleteByWebsiteID(ctx, website.ID); err != nil {
			return dto.Website{}, apperror.Wrap(
				apperror.ErrInternal.Code,
				apperror.ErrInternal.HTTPStatus,
				apperror.ErrInternal.Message,
				err,
			)
		}

		// Create new domains
		for _, domain := range *req.Domains {
			domainModel := &model.WebsiteDomain{
				WebsiteID: website.ID,
				Domain:    domain,
				IsPrimary: false,
			}
			if err := s.websiteDomainRepo.Create(ctx, domainModel); err != nil {
				return dto.Website{}, apperror.Wrap(
					apperror.ErrInternal.Code,
					apperror.ErrInternal.HTTPStatus,
					apperror.ErrInternal.Message,
					err,
				)
			}
		}
	}

	// Get updated website with domains
	return s.GetWebsite(ctx, id)
}

func (s *websiteService) DeleteWebsite(ctx context.Context, id int64) error {
	website, err := s.websiteRepo.GetByID(ctx, id)
	if err != nil {
		return apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if website == nil {
		return apperror.ErrWebsiteNotFound
	}

	// Delete domains first
	if err := s.websiteDomainRepo.DeleteByWebsiteID(ctx, id); err != nil {
		return apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	// Delete website
	if err := s.websiteRepo.Delete(ctx, id); err != nil {
		return apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	return nil
}

func (s *websiteService) EnableWebsite(ctx context.Context, id int64) error {
	return s.updateWebsiteStatus(ctx, id, 1)
}

func (s *websiteService) DisableWebsite(ctx context.Context, id int64) error {
	return s.updateWebsiteStatus(ctx, id, 0)
}

func (s *websiteService) updateWebsiteStatus(ctx context.Context, id int64, status int16) error {
	website, err := s.websiteRepo.GetByID(ctx, id)
	if err != nil {
		return apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if website == nil {
		return apperror.ErrWebsiteNotFound
	}

	website.Status = status
	if err := s.websiteRepo.Update(ctx, website); err != nil {
		return apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	return nil
}

func isValidRuntime(runtime string) bool {
	validRuntimes := map[string]bool{
		"php":     true,
		"node":    true,
		"python":  true,
		"static":  true,
	}
	return validRuntimes[runtime]
}
