package service

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/security"
)

type fakeSystemSettingRepo struct {
	mu       sync.Mutex
	nextID   int64
	settings map[string]*model.SystemSetting
}

func newFakeSystemSettingRepo() *fakeSystemSettingRepo {
	return &fakeSystemSettingRepo{
		nextID:   1,
		settings: map[string]*model.SystemSetting{},
	}
}

func (r *fakeSystemSettingRepo) GetByKey(_ context.Context, key string) (*model.SystemSetting, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	setting, ok := r.settings[key]
	if !ok {
		return nil, nil
	}
	cloned := *setting
	return &cloned, nil
}

func (r *fakeSystemSettingRepo) HasEncryptedSettings(_ context.Context) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, setting := range r.settings {
		if setting.IsEncrypted {
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeSystemSettingRepo) Upsert(_ context.Context, setting *model.SystemSetting) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cloned := *setting
	if existing, ok := r.settings[setting.Key]; ok {
		cloned.ID = existing.ID
		cloned.CreatedAt = existing.CreatedAt
	} else {
		cloned.ID = r.nextID
		r.nextID++
		cloned.CreatedAt = time.Now()
	}
	if cloned.UpdatedAt.IsZero() {
		cloned.UpdatedAt = time.Now()
	}
	r.settings[setting.Key] = &cloned
	setting.ID = cloned.ID
	setting.CreatedAt = cloned.CreatedAt
	setting.UpdatedAt = cloned.UpdatedAt
	return nil
}

func TestSystemSettingServiceEncryptsSensitiveValuesAtRest(t *testing.T) {
	repo := newFakeSystemSettingRepo()
	encryptor := testEncryptor(t)
	service := NewSystemSettingService(repo, encryptor)

	result, err := service.Upsert(context.Background(), dto.UpsertSystemSettingInput{
		Key:         "smtp.password",
		Value:       "plain-secret",
		IsEncrypted: true,
	})
	if err != nil {
		t.Fatalf("expected encrypted upsert success, got %v", err)
	}
	if result.Value != "plain-secret" {
		t.Fatalf("expected service result to expose plaintext to caller, got %q", result.Value)
	}

	stored := repo.settings["smtp.password"]
	if stored == nil {
		t.Fatal("expected setting to be persisted")
	}
	if !stored.IsEncrypted {
		t.Fatal("expected stored setting to be marked encrypted")
	}
	if strings.Contains(stored.Value, "plain-secret") {
		t.Fatalf("stored encrypted value leaked plaintext: %s", stored.Value)
	}

	loaded, err := service.GetByKey(context.Background(), "smtp.password")
	if err != nil {
		t.Fatalf("expected encrypted get success, got %v", err)
	}
	if loaded.Value != "plain-secret" {
		t.Fatalf("expected decrypted value, got %q", loaded.Value)
	}
}

func TestSystemSettingServiceRejectsEncryptedWriteWithoutKey(t *testing.T) {
	service := NewSystemSettingService(newFakeSystemSettingRepo(), nil)

	_, err := service.Upsert(context.Background(), dto.UpsertSystemSettingInput{
		Key:         "smtp.password",
		Value:       "plain-secret",
		IsEncrypted: true,
	})
	if err == nil {
		t.Fatal("expected encrypted write without key to fail")
	}
	if !strings.Contains(err.Error(), "SNOWPANEL_ENCRYPTION_KEY") {
		t.Fatalf("expected missing encryption key error, got %v", err)
	}
}

func TestSystemSettingServiceWrongKeyFailsSafely(t *testing.T) {
	repo := newFakeSystemSettingRepo()
	serviceA := NewSystemSettingService(repo, testEncryptor(t))
	if _, err := serviceA.Upsert(context.Background(), dto.UpsertSystemSettingInput{
		Key:         "smtp.password",
		Value:       "plain-secret",
		IsEncrypted: true,
	}); err != nil {
		t.Fatalf("expected encrypted upsert success, got %v", err)
	}

	serviceB := NewSystemSettingService(repo, testEncryptor(t))
	_, err := serviceB.GetByKey(context.Background(), "smtp.password")
	if err == nil {
		t.Fatal("expected wrong key decrypt to fail")
	}
	if strings.Contains(err.Error(), "plain-secret") {
		t.Fatalf("decrypt error leaked plaintext: %v", err)
	}
}

func TestSystemSettingServiceStoresPlainSettingWithoutEncryptor(t *testing.T) {
	repo := newFakeSystemSettingRepo()
	service := NewSystemSettingService(repo, nil)

	result, err := service.Upsert(context.Background(), dto.UpsertSystemSettingInput{
		Key:   "ui.theme",
		Value: "dark",
	})
	if err != nil {
		t.Fatalf("expected plain upsert success, got %v", err)
	}
	if result.Value != "dark" {
		t.Fatalf("unexpected result value: %s", result.Value)
	}
	if repo.settings["ui.theme"].Value != "dark" {
		t.Fatalf("expected plain value at rest for non-sensitive setting")
	}
}

func TestSystemSettingServiceRejectsEmptyKey(t *testing.T) {
	service := NewSystemSettingService(newFakeSystemSettingRepo(), nil)
	if _, err := service.Upsert(context.Background(), dto.UpsertSystemSettingInput{}); err == nil {
		t.Fatal("expected empty key upsert to fail")
	}
	if _, err := service.GetByKey(context.Background(), " "); err == nil {
		t.Fatal("expected empty key lookup to fail")
	}
}

func TestSystemSettingServiceMissingSetting(t *testing.T) {
	service := NewSystemSettingService(newFakeSystemSettingRepo(), nil)
	_, err := service.GetByKey(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected missing setting error")
	}
	appErr, ok := apperror.As(err)
	if !ok || appErr.Code != apperror.ErrSystemSettingNotFound.Code {
		t.Fatalf("expected system setting not found error, got %v", err)
	}
}

func testEncryptor(t *testing.T) *security.Encryptor {
	t.Helper()

	key, err := security.GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	encryptor, err := security.NewEncryptor(key, "test")
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}
	return encryptor
}
