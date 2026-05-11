package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/repository"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/security"
)

type AuditService interface {
	Record(ctx context.Context, input dto.RecordAuditInput)
	List(ctx context.Context, query dto.ListAuditLogsQuery) (dto.ListAuditLogsResult, error)
	Export(ctx context.Context, query dto.ListAuditLogsQuery, format string, writer io.Writer) error
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
		RequestSummary: security.RedactJSON(input.RequestSummary),
		Success:        input.Success,
		ResultCode:     input.ResultCode,
		ResultMessage:  security.RedactText(input.ResultMessage),
		RequestID:      input.RequestID,
		TraceID:        input.TraceID,
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

	filter, err := auditListFilter(query, page, size)
	if err != nil {
		return dto.ListAuditLogsResult{}, err
	}

	items, total, err := s.repo.List(ctx, filter)
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
			RequestID:      item.RequestID,
			TraceID:        item.TraceID,
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

func (s *auditService) Export(
	ctx context.Context,
	query dto.ListAuditLogsQuery,
	format string,
	writer io.Writer,
) error {
	normalizedFormat := strings.ToLower(strings.TrimSpace(format))
	if normalizedFormat == "" {
		normalizedFormat = "csv"
	}
	if normalizedFormat != "csv" && normalizedFormat != "jsonl" {
		return apperror.New(apperror.ErrBadRequest.Code, apperror.ErrBadRequest.HTTPStatus, "unsupported audit export format")
	}

	filter, err := auditListFilter(query, 1, 100)
	if err != nil {
		return err
	}

	switch normalizedFormat {
	case "csv":
		return s.exportCSV(ctx, filter, writer)
	case "jsonl":
		return s.exportJSONL(ctx, filter, writer)
	default:
		return apperror.New(apperror.ErrBadRequest.Code, apperror.ErrBadRequest.HTTPStatus, "unsupported audit export format")
	}
}

func (s *auditService) exportCSV(ctx context.Context, filter repository.AuditListFilter, writer io.Writer) error {
	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write([]string{
		"id",
		"user_id",
		"username",
		"ip",
		"module",
		"action",
		"target_type",
		"target_id",
		"success",
		"result_code",
		"request_id",
		"trace_id",
		"created_at",
		"request_summary",
		"result_message",
	}); err != nil {
		return err
	}

	err := s.paginateAuditExport(ctx, filter, func(item dto.AuditLog) error {
		userID := ""
		if item.UserID != nil {
			userID = fmt.Sprintf("%d", *item.UserID)
		}
		return csvWriter.Write([]string{
			fmt.Sprintf("%d", item.ID),
			userID,
			item.Username,
			item.IP,
			item.Module,
			item.Action,
			item.TargetType,
			item.TargetID,
			fmt.Sprintf("%t", item.Success),
			item.ResultCode,
			item.RequestID,
			item.TraceID,
			item.CreatedAt,
			item.RequestSummary,
			item.ResultMessage,
		})
	})
	if err != nil {
		return err
	}
	csvWriter.Flush()
	return csvWriter.Error()
}

func (s *auditService) exportJSONL(ctx context.Context, filter repository.AuditListFilter, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	return s.paginateAuditExport(ctx, filter, func(item dto.AuditLog) error {
		return encoder.Encode(item)
	})
}

func (s *auditService) paginateAuditExport(
	ctx context.Context,
	filter repository.AuditListFilter,
	writeItem func(dto.AuditLog) error,
) error {
	const maxRows = 100000
	exported := 0
	page := 1
	for {
		filter.Page = page
		filter.Size = 100
		items, _, err := s.repo.List(ctx, filter)
		if err != nil {
			return apperror.Wrap(
				apperror.ErrInternal.Code,
				apperror.ErrInternal.HTTPStatus,
				apperror.ErrInternal.Message,
				err,
			)
		}
		if len(items) == 0 {
			return nil
		}
		for _, item := range items {
			if exported >= maxRows {
				return nil
			}
			if err := writeItem(toAuditLogDTO(item)); err != nil {
				return err
			}
			exported++
		}
		if len(items) < filter.Size {
			return nil
		}
		page++
	}
}

func auditListFilter(
	query dto.ListAuditLogsQuery,
	page int,
	size int,
) (repository.AuditListFilter, error) {
	var startTime *time.Time
	if strings.TrimSpace(query.StartTime) != "" {
		parsed, err := parseAuditTime(query.StartTime, false)
		if err != nil {
			return repository.AuditListFilter{}, apperror.New(apperror.ErrBadRequest.Code, apperror.ErrBadRequest.HTTPStatus, "invalid start_time")
		}
		startTime = &parsed
	}
	var endTime *time.Time
	if strings.TrimSpace(query.EndTime) != "" {
		parsed, err := parseAuditTime(query.EndTime, true)
		if err != nil {
			return repository.AuditListFilter{}, apperror.New(apperror.ErrBadRequest.Code, apperror.ErrBadRequest.HTTPStatus, "invalid end_time")
		}
		endTime = &parsed
	}
	var userID *int64
	if query.UserID > 0 {
		userID = &query.UserID
	}
	return repository.AuditListFilter{
		Page:       page,
		Size:       size,
		StartTime:  startTime,
		EndTime:    endTime,
		UserID:     userID,
		Username:   strings.TrimSpace(query.Username),
		Module:     strings.TrimSpace(query.Module),
		Action:     strings.TrimSpace(query.Action),
		TargetType: strings.TrimSpace(query.TargetType),
		TargetID:   strings.TrimSpace(query.TargetID),
		Success:    query.Success,
		ResultCode: strings.TrimSpace(query.ResultCode),
		RequestID:  strings.TrimSpace(query.RequestID),
		TraceID:    strings.TrimSpace(query.TraceID),
	}, nil
}

func parseAuditTime(raw string, endOfDay bool) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, err
	}
	if endOfDay {
		return parsed.Add(24*time.Hour - time.Nanosecond), nil
	}
	return parsed, nil
}

func toAuditLogDTO(item model.AuditLog) dto.AuditLog {
	return dto.AuditLog{
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
		RequestID:      item.RequestID,
		TraceID:        item.TraceID,
		CreatedAt:      item.CreatedAt.Format(time.RFC3339),
	}
}
