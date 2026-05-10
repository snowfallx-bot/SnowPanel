package main

import (
	"context"
	"strings"
	"testing"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/security"
)

type startupSystemSettingRepo struct {
	hasEncrypted bool
}

func (r startupSystemSettingRepo) GetByKey(context.Context, string) (*model.SystemSetting, error) {
	return nil, nil
}

func (r startupSystemSettingRepo) HasEncryptedSettings(context.Context) (bool, error) {
	return r.hasEncrypted, nil
}

func (r startupSystemSettingRepo) Upsert(context.Context, *model.SystemSetting) error {
	return nil
}

func TestValidateEncryptedSettingsStartupAllowsEmptyDatabaseWithoutKey(t *testing.T) {
	err := validateEncryptedSettingsStartup(
		context.Background(),
		startupSystemSettingRepo{hasEncrypted: false},
		nil,
	)
	if err != nil {
		t.Fatalf("expected startup validation success, got %v", err)
	}
}

func TestValidateEncryptedSettingsStartupRejectsEncryptedSettingsWithoutKey(t *testing.T) {
	err := validateEncryptedSettingsStartup(
		context.Background(),
		startupSystemSettingRepo{hasEncrypted: true},
		nil,
	)
	if err == nil {
		t.Fatal("expected startup validation failure")
	}
	if !strings.Contains(err.Error(), "SNOWPANEL_ENCRYPTION_KEY") {
		t.Fatalf("expected missing encryption key error, got %v", err)
	}
}

func TestValidateEncryptedSettingsStartupAllowsEncryptedSettingsWithKey(t *testing.T) {
	key, err := security.GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	encryptor, err := security.NewEncryptor(key, "test")
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	err = validateEncryptedSettingsStartup(
		context.Background(),
		startupSystemSettingRepo{hasEncrypted: true},
		encryptor,
	)
	if err != nil {
		t.Fatalf("expected startup validation success, got %v", err)
	}
}
