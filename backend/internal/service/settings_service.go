package service

import (
	"context"
	"time"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/repository"
)

type SettingsService interface {
	ListSettings(ctx context.Context) ([]dto.SettingItem, error)
	GetSetting(ctx context.Context, key string) (dto.SettingItem, error)
	CreateSetting(ctx context.Context, req dto.CreateSettingRequest) (dto.SettingItem, error)
	UpdateSetting(ctx context.Context, key string, req dto.UpdateSettingRequest) (dto.SettingItem, error)
	DeleteSetting(ctx context.Context, key string) error
}

type settingsService struct {
	repo repository.SettingsRepository
}

func NewSettingsService(repo repository.SettingsRepository) SettingsService {
	return &settingsService{repo: repo}
}

func (s *settingsService) ListSettings(ctx context.Context) ([]dto.SettingItem, error) {
	items, err := s.repo.List(ctx)
	if err != nil {
		return nil, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	result := make([]dto.SettingItem, 0, len(items))
	for _, item := range items {
		result = append(result, s.mapSetting(item))
	}

	return result, nil
}

func (s *settingsService) GetSetting(ctx context.Context, key string) (dto.SettingItem, error) {
	setting, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return dto.SettingItem{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if setting == nil {
		return dto.SettingItem{}, apperror.ErrSettingNotFound
	}

	return s.mapSetting(*setting), nil
}

func (s *settingsService) CreateSetting(ctx context.Context, req dto.CreateSettingRequest) (dto.SettingItem, error) {
	// Validate key format
	if !isValidSettingKey(req.Key) {
		return dto.SettingItem{}, apperror.ErrSettingKeyInvalid
	}

	// Check if key already exists
	existing, err := s.repo.GetByKey(ctx, req.Key)
	if err != nil {
		return dto.SettingItem{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if existing != nil {
		return dto.SettingItem{}, apperror.ErrSettingKeyExists
	}

	setting := &model.SystemSetting{
		Key:         req.Key,
		Value:       req.Value,
		ValueType:   req.ValueType,
		Description: req.Description,
		IsEncrypted: req.IsEncrypted,
	}

	if err := s.repo.Create(ctx, setting); err != nil {
		return dto.SettingItem{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	return s.mapSetting(*setting), nil
}

func (s *settingsService) UpdateSetting(ctx context.Context, key string, req dto.UpdateSettingRequest) (dto.SettingItem, error) {
	setting, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return dto.SettingItem{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if setting == nil {
		return dto.SettingItem{}, apperror.ErrSettingNotFound
	}

	// Update fields only if provided
	if req.Value != nil {
		setting.Value = *req.Value
	}
	if req.ValueType != nil {
		setting.ValueType = *req.ValueType
	}
	if req.Description != nil {
		setting.Description = *req.Description
	}
	if req.IsEncrypted != nil {
		setting.IsEncrypted = *req.IsEncrypted
	}

	if err := s.repo.Update(ctx, setting); err != nil {
		return dto.SettingItem{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	return s.mapSetting(*setting), nil
}

func (s *settingsService) DeleteSetting(ctx context.Context, key string) error {
	if err := s.repo.Delete(ctx, key); err != nil {
		return apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	return nil
}

func (s *settingsService) mapSetting(setting model.SystemSetting) dto.SettingItem {
	return dto.SettingItem{
		Key:         setting.Key,
		Value:       trimSensitiveValue(setting.Value, setting.IsEncrypted),
		ValueType:   setting.ValueType,
		IsEncrypted: setting.IsEncrypted,
		Description: setting.Description,
		CreatedAt:   setting.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   setting.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// trimSensitiveValue returns "***" for encrypted values to avoid leaking sensitive data
func trimSensitiveValue(value string, isEncrypted bool) string {
	if isEncrypted {
		if len(value) == 0 {
			return ""
		}
		return "********"
	}
	return value
}

// isValidSettingKey validates the setting key format
func isValidSettingKey(key string) bool {
	if len(key) == 0 {
		return false
	}
	// Only allow alphanumeric, underscores, hyphens, and dots
	for _, r := range key {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.') {
			return false
		}
	}
	return true
}
