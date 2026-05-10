package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/repository"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/security"
)

type SystemSettingService interface {
	Upsert(ctx context.Context, input dto.UpsertSystemSettingInput) (dto.SystemSetting, error)
	GetByKey(ctx context.Context, key string) (dto.SystemSetting, error)
}

type systemSettingService struct {
	repo      repository.SystemSettingRepository
	encryptor *security.Encryptor
}

func NewSystemSettingService(
	repo repository.SystemSettingRepository,
	encryptor *security.Encryptor,
) SystemSettingService {
	return &systemSettingService{
		repo:      repo,
		encryptor: encryptor,
	}
}

func (s *systemSettingService) Upsert(
	ctx context.Context,
	input dto.UpsertSystemSettingInput,
) (dto.SystemSetting, error) {
	key := strings.TrimSpace(input.Key)
	if key == "" {
		return dto.SystemSetting{}, apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			errors.New("setting key is required"),
		)
	}

	value := input.Value
	if input.IsEncrypted {
		if s.encryptor == nil {
			return dto.SystemSetting{}, apperror.Wrap(
				apperror.ErrInternal.Code,
				apperror.ErrInternal.HTTPStatus,
				apperror.ErrInternal.Message,
				errors.New("SNOWPANEL_ENCRYPTION_KEY cannot be empty when encrypted settings are used"),
			)
		}
		encrypted, err := s.encryptor.Encrypt(input.Value)
		if err != nil {
			return dto.SystemSetting{}, apperror.Wrap(
				apperror.ErrInternal.Code,
				apperror.ErrInternal.HTTPStatus,
				apperror.ErrInternal.Message,
				err,
			)
		}
		value = encrypted
	}

	now := time.Now()
	setting := &model.SystemSetting{
		Key:         key,
		Value:       value,
		ValueType:   normalizeSettingValueType(input.ValueType),
		IsEncrypted: input.IsEncrypted,
		Description: strings.TrimSpace(input.Description),
		UpdatedBy:   input.UpdatedBy,
		UpdatedAt:   now,
	}
	if err := s.repo.Upsert(ctx, setting); err != nil {
		return dto.SystemSetting{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	setting.Value = input.Value
	return mapSystemSetting(*setting), nil
}

func (s *systemSettingService) GetByKey(ctx context.Context, key string) (dto.SystemSetting, error) {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return dto.SystemSetting{}, apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			apperror.ErrBadRequest.Message,
			errors.New("setting key is required"),
		)
	}

	setting, err := s.repo.GetByKey(ctx, trimmed)
	if err != nil {
		return dto.SystemSetting{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if setting == nil {
		return dto.SystemSetting{}, apperror.Wrap(
			apperror.ErrSystemSettingNotFound.Code,
			apperror.ErrSystemSettingNotFound.HTTPStatus,
			apperror.ErrSystemSettingNotFound.Message,
			errors.New("system setting not found"),
		)
	}

	if setting.IsEncrypted {
		if s.encryptor == nil {
			return dto.SystemSetting{}, apperror.Wrap(
				apperror.ErrInternal.Code,
				apperror.ErrInternal.HTTPStatus,
				apperror.ErrInternal.Message,
				errors.New("SNOWPANEL_ENCRYPTION_KEY cannot be empty when encrypted settings are used"),
			)
		}
		value, err := s.encryptor.Decrypt(setting.Value)
		if err != nil {
			return dto.SystemSetting{}, apperror.Wrap(
				apperror.ErrInternal.Code,
				apperror.ErrInternal.HTTPStatus,
				apperror.ErrInternal.Message,
				err,
			)
		}
		setting.Value = value
	}

	return mapSystemSetting(*setting), nil
}

func normalizeSettingValueType(raw string) string {
	normalized := strings.TrimSpace(strings.ToLower(raw))
	if normalized == "" {
		return "string"
	}
	return normalized
}

func mapSystemSetting(setting model.SystemSetting) dto.SystemSetting {
	return dto.SystemSetting{
		ID:          setting.ID,
		Key:         setting.Key,
		Value:       setting.Value,
		ValueType:   setting.ValueType,
		IsEncrypted: setting.IsEncrypted,
		Description: setting.Description,
		UpdatedBy:   setting.UpdatedBy,
		CreatedAt:   setting.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   setting.UpdatedAt.Format(time.RFC3339),
	}
}
