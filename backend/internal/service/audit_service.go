package service

import (
	"context"
	"time"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/repository"
)

type AuditService interface {
	Record(ctx context.Context, input dto.RecordAuditInput)
	List(ctx context.Context, query dto.ListAuditLogsQuery) (dto.ListAuditLogsResult, error)
}

type auditService struct {
	repo repository.AuditRepository
}

func NewAuditService(repo repository.AuditRepository) AuditService {
	return &auditService{
		repo: repo,
	}
}

func (s *auditService) Record(ctx context.Context, input dto.RecordAuditInput) {
	item := &model.AuditLog{
		UserID:         input.UserID,
		Username:       input.Username,
		IP:             input.IP,
		Module:         input.Module,
		Action:         input.Action,
		TargetType:     input.TargetType,
		TargetID:       input.TargetID,
		RequestSummary: input.RequestSummary,
		Success:        input.Success,
		ResultCode:     input.ResultCode,
		ResultMessage:  input.ResultMessage,
	}
	_ = s.repo.Create(ctx, item)
}

func (s *auditService) List(
	ctx context.Context,
	query dto.ListAuditLogsQuery,
) (dto.ListAuditLogsResult, error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	size := query.Size
	if size <= 0 {
		size = 20
	}
	if size > 100 {
		size = 100
	}

	items, total, err := s.repo.List(ctx, repository.AuditListFilter{
		Page:   page,
		Size:   size,
		Module: query.Module,
		Action: query.Action,
	})
	if err != nil {
		return dto.ListAuditLogsResult{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	result := make([]dto.AuditLog, 0, len(items))
	for _, item := range items {
		result = append(result, dto.AuditLog{
			ID:             item.ID,
			UserID:         item.UserID,
			Username:       item.Username,
			IP:             item.IP,
			Module:         item.Module,
			Action:         item.Action,
			TargetType:     item.TargetType,
			TargetID:       item.TargetID,
			RequestSummary: item.RequestSummary,
			Success:        item.Success,
			ResultCode:     item.ResultCode,
			ResultMessage:  item.ResultMessage,
			CreatedAt:      item.CreatedAt.Format(time.RFC3339),
		})
	}

	return dto.ListAuditLogsResult{
		Page:  page,
		Size:  size,
		Total: total,
		Items: result,
	}, nil
}
