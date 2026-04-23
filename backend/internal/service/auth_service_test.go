package service

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/config"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"golang.org/x/crypto/bcrypt"
)

type fakeUserRepo struct {
	count           int64
	countErr        error
	createErr       error
	getByNameErr    error
	getByIDErr      error
	updateLoginErr  error
	updatePassErr   error
	createdUser     *model.User
	usersByName     map[string]*model.User
	usersByID       map[int64]*model.User
	userRoles       map[int64][]string
	rolePermissions map[string][]string
	updatedLogins   map[int64]time.Time
	updatedHashes   map[int64]string
}

func (r *fakeUserRepo) Count(context.Context) (int64, error) {
	return r.count, r.countErr
}

func (r *fakeUserRepo) Create(_ context.Context, user *model.User) error {
	if r.createErr != nil {
		return r.createErr
	}
	cloned := *user
	r.createdUser = &cloned
	if r.usersByName == nil {
		r.usersByName = map[string]*model.User{}
	}
	if r.usersByID == nil {
		r.usersByID = map[int64]*model.User{}
	}
	r.usersByName[user.Username] = &cloned
	if user.ID != 0 {
		r.usersByID[user.ID] = &cloned
	}
	return nil
}

func (r *fakeUserRepo) GetByUsername(_ context.Context, username string) (*model.User, error) {
	if r.getByNameErr != nil {
		return nil, r.getByNameErr
	}
	user, ok := r.usersByName[username]
	if !ok {
		return nil, nil
	}
	cloned := *user
	return &cloned, nil
}

func (r *fakeUserRepo) GetByID(_ context.Context, id int64) (*model.User, error) {
	if r.getByIDErr != nil {
		return nil, r.getByIDErr
	}
	user, ok := r.usersByID[id]
	if !ok {
		return nil, nil
	}
	cloned := *user
	return &cloned, nil
}

func (r *fakeUserRepo) UpdateLastLoginAt(_ context.Context, id int64, at time.Time) error {
	if r.updateLoginErr != nil {
		return r.updateLoginErr
	}
	if r.updatedLogins == nil {
		r.updatedLogins = map[int64]time.Time{}
	}
	r.updatedLogins[id] = at

	if user, ok := r.usersByID[id]; ok {
		cloned := *user
		cloned.LastLoginAt = &at
		r.usersByID[id] = &cloned
		if r.usersByName != nil {
			r.usersByName[cloned.Username] = &cloned
		}
	}
	return nil
}

func (r *fakeUserRepo) UpdatePasswordHash(_ context.Context, id int64, hash string) error {
	if r.updatePassErr != nil {
		return r.updatePassErr
	}
	if r.updatedHashes == nil {
		r.updatedHashes = map[int64]string{}
	}
	r.updatedHashes[id] = hash

	if user, ok := r.usersByID[id]; ok {
		cloned := *user
		cloned.PasswordHash = hash
		r.usersByID[id] = &cloned
		if r.usersByName != nil {
			r.usersByName[cloned.Username] = &cloned
		}
	}
	return nil
}

func (r *fakeUserRepo) EnsureRBACDefaults(context.Context) error {
	if r.rolePermissions == nil {
		r.rolePermissions = map[string][]string{}
	}
	r.rolePermissions["super_admin"] = []string{
		"dashboard.read",
		"files.read",
		"files.write",
		"services.read",
		"services.manage",
		"docker.read",
		"docker.manage",
		"cron.read",
		"cron.manage",
		"audit.read",
		"tasks.read",
		"tasks.manage",
	}
	r.rolePermissions["operator"] = []string{
		"dashboard.read",
		"files.read",
	}
	return nil
}

func (r *fakeUserRepo) EnsureUserRoleBySlug(_ context.Context, userID int64, roleSlug string) error {
	if r.userRoles == nil {
		r.userRoles = map[int64][]string{}
	}
	roles := r.userRoles[userID]
	if !slices.Contains(roles, roleSlug) {
		r.userRoles[userID] = append(roles, roleSlug)
	}
	return nil
}

func (r *fakeUserRepo) GetRolesAndPermissions(_ context.Context, userID int64) ([]string, []string, error) {
	roles := append([]string(nil), r.userRoles[userID]...)
	slices.Sort(roles)
	permissionSet := map[string]struct{}{}
	for _, role := range roles {
		for _, permission := range r.rolePermissions[role] {
			permissionSet[permission] = struct{}{}
		}
	}

	permissions := make([]string, 0, len(permissionSet))
	for permission := range permissionSet {
		permissions = append(permissions, permission)
	}
	slices.Sort(permissions)
	return roles, permissions, nil
}

func testAuthConfig() config.AuthConfig {
	return config.AuthConfig{
		AppEnv:               "development",
		JWTSecret:            "unit-test-secret",
		JWTIssuer:            "snowpanel-test",
		JWTExpire:            2 * time.Hour,
		JWTRefreshExpire:     7 * 24 * time.Hour,
		BootstrapAdmin:       true,
		DefaultAdminUsername: "admin",
		DefaultAdminEmail:    "admin@example.com",
		DefaultAdminPassword: "admin123456",
	}
}

func TestEnsureDefaultAdminRejectsWeakPasswordInProduction(t *testing.T) {
	repo := &fakeUserRepo{
		count:       0,
		usersByName: map[string]*model.User{},
		usersByID:   map[int64]*model.User{},
	}

	cfg := testAuthConfig()
	cfg.AppEnv = "production"
	cfg.DefaultAdminPassword = "admin123456"

	service := NewAuthService(repo, cfg)
	err := service.EnsureDefaultAdmin(context.Background())
	if err == nil {
		t.Fatalf("expected error for weak bootstrap password in production")
	}
}

func TestEnsureDefaultAdminGeneratesPasswordInDevelopmentWhenMissing(t *testing.T) {
	repo := &fakeUserRepo{
		count:       0,
		usersByName: map[string]*model.User{},
		usersByID:   map[int64]*model.User{},
	}

	cfg := testAuthConfig()
	cfg.AppEnv = "development"
	cfg.DefaultAdminPassword = ""

	service := NewAuthService(repo, cfg)
	if err := service.EnsureDefaultAdmin(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.createdUser == nil {
		t.Fatalf("expected default admin to be created")
	}
	if repo.createdUser.PasswordHash == "" {
		t.Fatalf("expected generated password hash")
	}
}

func TestEnsureDefaultAdminCreatesUserWhenDatabaseEmpty(t *testing.T) {
	repo := &fakeUserRepo{
		count:       0,
		usersByName: map[string]*model.User{},
		usersByID:   map[int64]*model.User{},
	}
	service := NewAuthService(repo, testAuthConfig())

	if err := service.EnsureDefaultAdmin(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.createdUser == nil {
		t.Fatalf("expected default admin to be created")
	}
	if repo.createdUser.Username != "admin" {
		t.Fatalf("unexpected username: %s", repo.createdUser.Username)
	}
	if repo.createdUser.Email != "admin@example.com" {
		t.Fatalf("unexpected email: %s", repo.createdUser.Email)
	}
	if repo.createdUser.PasswordHash == "admin123456" {
		t.Fatalf("password hash should not equal plain text")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(repo.createdUser.PasswordHash), []byte("admin123456")); err != nil {
		t.Fatalf("expected password hash to be valid bcrypt hash: %v", err)
	}
}

func TestEnsureDefaultAdminSkipsWhenUsersAlreadyExist(t *testing.T) {
	repo := &fakeUserRepo{
		count:       1,
		usersByName: map[string]*model.User{},
		usersByID:   map[int64]*model.User{},
	}
	service := NewAuthService(repo, testAuthConfig())

	if err := service.EnsureDefaultAdmin(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.createdUser != nil {
		t.Fatalf("did not expect admin creation when users already exist")
	}
}

func TestEnsureDefaultAdminWrapsRepositoryError(t *testing.T) {
	repo := &fakeUserRepo{countErr: errors.New("count failed")}
	service := NewAuthService(repo, testAuthConfig())

	err := service.EnsureDefaultAdmin(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != apperror.ErrBootstrapAdminFail.Code {
		t.Fatalf("unexpected error code: %d", appErr.Code)
	}
}

func TestLoginSuccessAndParseToken(t *testing.T) {
	password := "StrongPassword#1"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate bcrypt hash: %v", err)
	}

	repo := &fakeUserRepo{
		usersByName: map[string]*model.User{
			"admin": {
				ID:           7,
				Username:     "admin",
				Email:        "admin@example.com",
				PasswordHash: string(hash),
				Status:       1,
			},
		},
		usersByID: map[int64]*model.User{
			7: {
				ID:           7,
				Username:     "admin",
				Email:        "admin@example.com",
				PasswordHash: string(hash),
				Status:       1,
			},
		},
		userRoles: map[int64][]string{
			7: []string{"super_admin"},
		},
	}
	if err := repo.EnsureRBACDefaults(context.Background()); err != nil {
		t.Fatalf("failed to seed fake rbac defaults: %v", err)
	}
	service := NewAuthService(repo, testAuthConfig())

	result, err := service.Login(context.Background(), dto.LoginRequest{
		Username: "admin",
		Password: password,
	})
	if err != nil {
		t.Fatalf("expected login success, got error: %v", err)
	}
	if result.AccessToken == "" {
		t.Fatalf("expected non-empty access token")
	}
	if result.RefreshToken == "" {
		t.Fatalf("expected non-empty refresh token")
	}
	if result.RefreshExpiresIn <= result.ExpiresIn {
		t.Fatalf("expected refresh token lifetime to be longer than access token")
	}
	if result.User.Username != "admin" {
		t.Fatalf("unexpected user profile username: %s", result.User.Username)
	}
	if !result.User.MustChangePassword {
		t.Fatalf("expected must_change_password=true for first bootstrap admin login")
	}

	claims, err := service.ParseToken(result.AccessToken)
	if err != nil {
		t.Fatalf("expected token to be parseable, got %v", err)
	}
	if claims.UserID != 7 {
		t.Fatalf("unexpected claim user id: %d", claims.UserID)
	}
	if claims.Username != "admin" {
		t.Fatalf("unexpected claim username: %s", claims.Username)
	}
	if !slices.Contains(claims.Roles, "super_admin") {
		t.Fatalf("expected super_admin role in token claims")
	}
	if !slices.Contains(claims.Permissions, "docker.manage") {
		t.Fatalf("expected docker.manage permission in token claims")
	}
	if !claims.MustChangePassword {
		t.Fatalf("expected must_change_password=true in token claims")
	}
	if claims.SessionIssuedAt <= 0 {
		t.Fatalf("expected session_issued_at claim to be set")
	}
	if _, exists := repo.updatedLogins[7]; exists {
		t.Fatalf("did not expect last_login_at update when password change is required")
	}
}

func TestLoginRejectsInvalidCredential(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("expected"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate bcrypt hash: %v", err)
	}

	repo := &fakeUserRepo{
		usersByName: map[string]*model.User{
			"operator": {
				ID:           9,
				Username:     "operator",
				PasswordHash: string(hash),
				Status:       1,
			},
		},
		usersByID: map[int64]*model.User{},
	}
	service := NewAuthService(repo, testAuthConfig())

	_, err = service.Login(context.Background(), dto.LoginRequest{
		Username: "operator",
		Password: "wrong-password",
	})
	if err == nil {
		t.Fatalf("expected login failure")
	}
	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != apperror.ErrInvalidCredential.Code {
		t.Fatalf("unexpected error code: %d", appErr.Code)
	}
}

func TestLoginRejectsDisabledUser(t *testing.T) {
	password := "StrongPassword#1"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate bcrypt hash: %v", err)
	}

	repo := &fakeUserRepo{
		usersByName: map[string]*model.User{
			"disabled": {
				ID:           10,
				Username:     "disabled",
				PasswordHash: string(hash),
				Status:       0,
			},
		},
		usersByID: map[int64]*model.User{
			10: {
				ID:           10,
				Username:     "disabled",
				PasswordHash: string(hash),
				Status:       0,
			},
		},
	}
	service := NewAuthService(repo, testAuthConfig())

	_, err = service.Login(context.Background(), dto.LoginRequest{
		Username: "disabled",
		Password: password,
	})
	if err == nil {
		t.Fatalf("expected login failure for disabled user")
	}
	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != apperror.ErrUserDisabled.Code {
		t.Fatalf("unexpected error code: %d", appErr.Code)
	}
}

func TestLoginUpdatesLastLoginForNonBootstrapUser(t *testing.T) {
	password := "StrongPassword#1"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate bcrypt hash: %v", err)
	}

	repo := &fakeUserRepo{
		usersByName: map[string]*model.User{
			"operator": {
				ID:           9,
				Username:     "operator",
				Email:        "operator@example.com",
				PasswordHash: string(hash),
				Status:       1,
			},
		},
		usersByID: map[int64]*model.User{
			9: {
				ID:           9,
				Username:     "operator",
				Email:        "operator@example.com",
				PasswordHash: string(hash),
				Status:       1,
			},
		},
		userRoles: map[int64][]string{
			9: []string{"operator"},
		},
	}
	if err := repo.EnsureRBACDefaults(context.Background()); err != nil {
		t.Fatalf("failed to seed fake rbac defaults: %v", err)
	}

	service := NewAuthService(repo, testAuthConfig())
	result, err := service.Login(context.Background(), dto.LoginRequest{
		Username: "operator",
		Password: password,
	})
	if err != nil {
		t.Fatalf("expected login success, got %v", err)
	}
	if result.User.MustChangePassword {
		t.Fatalf("did not expect must_change_password for operator")
	}
	if _, exists := repo.updatedLogins[9]; !exists {
		t.Fatalf("expected last_login_at to be updated for operator")
	}

	claims, err := service.ParseToken(result.AccessToken)
	if err != nil {
		t.Fatalf("expected token parse success, got %v", err)
	}
	if claims.MustChangePassword {
		t.Fatalf("did not expect must_change_password claim for operator")
	}
	if claims.SessionIssuedAt <= 0 {
		t.Fatalf("expected session_issued_at claim to be set")
	}
}

func TestChangePasswordSuccessClearsMustChangeFlag(t *testing.T) {
	oldPassword := "StrongPassword#1"
	newPassword := "EvenStronger#2"
	hash, err := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate bcrypt hash: %v", err)
	}

	repo := &fakeUserRepo{
		usersByName: map[string]*model.User{
			"admin": {
				ID:           7,
				Username:     "admin",
				Email:        "admin@example.com",
				PasswordHash: string(hash),
				Status:       1,
			},
		},
		usersByID: map[int64]*model.User{
			7: {
				ID:           7,
				Username:     "admin",
				Email:        "admin@example.com",
				PasswordHash: string(hash),
				Status:       1,
			},
		},
		userRoles: map[int64][]string{
			7: []string{"super_admin"},
		},
	}
	if err := repo.EnsureRBACDefaults(context.Background()); err != nil {
		t.Fatalf("failed to seed fake rbac defaults: %v", err)
	}

	service := NewAuthService(repo, testAuthConfig())
	resp, err := service.ChangePassword(context.Background(), 7, dto.ChangePasswordRequest{
		CurrentPassword: oldPassword,
		NewPassword:     newPassword,
	})
	if err != nil {
		t.Fatalf("expected change password success, got %v", err)
	}
	if resp.User.MustChangePassword {
		t.Fatalf("expected must_change_password=false after password change")
	}
	updatedHash := repo.updatedHashes[7]
	if updatedHash == "" {
		t.Fatalf("expected password hash update to be persisted")
	}
	if bcrypt.CompareHashAndPassword([]byte(updatedHash), []byte(newPassword)) != nil {
		t.Fatalf("expected updated hash to match new password")
	}
	if _, exists := repo.updatedLogins[7]; !exists {
		t.Fatalf("expected last_login_at update after password change")
	}

	claims, err := service.ParseToken(resp.AccessToken)
	if err != nil {
		t.Fatalf("expected token parse success, got %v", err)
	}
	if claims.MustChangePassword {
		t.Fatalf("expected must_change_password=false in new token")
	}
	if claims.SessionIssuedAt <= 0 {
		t.Fatalf("expected session_issued_at claim to be set")
	}
}

func TestValidateSessionRejectsExpiredSessionByLastLoginAt(t *testing.T) {
	lastLogin := time.Now().UTC()
	repo := &fakeUserRepo{
		usersByID: map[int64]*model.User{
			7: {
				ID:          7,
				Username:    "admin",
				Email:       "admin@example.com",
				Status:      1,
				LastLoginAt: &lastLogin,
			},
		},
	}
	service := NewAuthService(repo, testAuthConfig())
	err := service.ValidateSession(context.Background(), TokenClaims{
		UserID:          7,
		SessionIssuedAt: lastLogin.Add(-time.Second).UnixNano(),
	})
	if err == nil {
		t.Fatalf("expected session validation failure")
	}
	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != apperror.ErrSessionExpired.Code {
		t.Fatalf("unexpected error code: %d", appErr.Code)
	}
}

func TestValidateSessionRejectsDisabledUser(t *testing.T) {
	lastLogin := time.Now().UTC()
	repo := &fakeUserRepo{
		usersByID: map[int64]*model.User{
			7: {
				ID:          7,
				Username:    "admin",
				Email:       "admin@example.com",
				Status:      0,
				LastLoginAt: &lastLogin,
			},
		},
	}
	service := NewAuthService(repo, testAuthConfig())
	err := service.ValidateSession(context.Background(), TokenClaims{
		UserID:          7,
		SessionIssuedAt: lastLogin.UnixNano(),
	})
	if err == nil {
		t.Fatalf("expected session validation failure for disabled user")
	}
	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != apperror.ErrSessionExpired.Code {
		t.Fatalf("unexpected error code: %d", appErr.Code)
	}
}

func TestValidateSessionAcceptsFreshSession(t *testing.T) {
	lastLogin := time.Now().UTC()
	repo := &fakeUserRepo{
		usersByID: map[int64]*model.User{
			7: {
				ID:          7,
				Username:    "admin",
				Email:       "admin@example.com",
				Status:      1,
				LastLoginAt: &lastLogin,
			},
		},
	}
	service := NewAuthService(repo, testAuthConfig())
	err := service.ValidateSession(context.Background(), TokenClaims{
		UserID:          7,
		SessionIssuedAt: lastLogin.UnixNano(),
	})
	if err != nil {
		t.Fatalf("expected session validation success, got %v", err)
	}
}

func TestParseTokenRejectsRefreshToken(t *testing.T) {
	password := "StrongPassword#1"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate bcrypt hash: %v", err)
	}

	repo := &fakeUserRepo{
		usersByName: map[string]*model.User{
			"operator": {
				ID:           9,
				Username:     "operator",
				Email:        "operator@example.com",
				PasswordHash: string(hash),
				Status:       1,
			},
		},
		usersByID: map[int64]*model.User{
			9: {
				ID:           9,
				Username:     "operator",
				Email:        "operator@example.com",
				PasswordHash: string(hash),
				Status:       1,
			},
		},
		userRoles: map[int64][]string{
			9: []string{"operator"},
		},
	}
	if err := repo.EnsureRBACDefaults(context.Background()); err != nil {
		t.Fatalf("failed to seed fake rbac defaults: %v", err)
	}

	service := NewAuthService(repo, testAuthConfig())
	result, err := service.Login(context.Background(), dto.LoginRequest{
		Username: "operator",
		Password: password,
	})
	if err != nil {
		t.Fatalf("expected login success, got %v", err)
	}

	_, err = service.ParseToken(result.RefreshToken)
	if err == nil {
		t.Fatalf("expected ParseToken to reject refresh token")
	}
	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != apperror.ErrTokenParse.Code {
		t.Fatalf("unexpected error code: %d", appErr.Code)
	}
}

func TestRefreshRotatesSessionAndReturnsNewTokenPair(t *testing.T) {
	password := "StrongPassword#1"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate bcrypt hash: %v", err)
	}

	repo := &fakeUserRepo{
		usersByName: map[string]*model.User{
			"operator": {
				ID:           9,
				Username:     "operator",
				Email:        "operator@example.com",
				PasswordHash: string(hash),
				Status:       1,
			},
		},
		usersByID: map[int64]*model.User{
			9: {
				ID:           9,
				Username:     "operator",
				Email:        "operator@example.com",
				PasswordHash: string(hash),
				Status:       1,
			},
		},
		userRoles: map[int64][]string{
			9: []string{"operator"},
		},
	}
	if err := repo.EnsureRBACDefaults(context.Background()); err != nil {
		t.Fatalf("failed to seed fake rbac defaults: %v", err)
	}

	service := NewAuthService(repo, testAuthConfig())
	loginResp, err := service.Login(context.Background(), dto.LoginRequest{
		Username: "operator",
		Password: password,
	})
	if err != nil {
		t.Fatalf("expected login success, got %v", err)
	}

	refreshResp, err := service.Refresh(context.Background(), loginResp.RefreshToken)
	if err != nil {
		t.Fatalf("expected refresh success, got %v", err)
	}
	if refreshResp.AccessToken == "" || refreshResp.RefreshToken == "" {
		t.Fatalf("expected refreshed access+refresh tokens")
	}
	if refreshResp.AccessToken == loginResp.AccessToken {
		t.Fatalf("expected refreshed access token to rotate")
	}
	if refreshResp.RefreshToken == loginResp.RefreshToken {
		t.Fatalf("expected refreshed refresh token to rotate")
	}

	// Previous refresh token should be revoked by session rotation.
	_, err = service.Refresh(context.Background(), loginResp.RefreshToken)
	if err == nil {
		t.Fatalf("expected old refresh token to be rejected after rotation")
	}
	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != apperror.ErrSessionExpired.Code {
		t.Fatalf("unexpected error code: %d", appErr.Code)
	}
}

func TestRevokeSessionInvalidatesExistingAccessToken(t *testing.T) {
	password := "StrongPassword#1"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate bcrypt hash: %v", err)
	}

	repo := &fakeUserRepo{
		usersByName: map[string]*model.User{
			"operator": {
				ID:           9,
				Username:     "operator",
				Email:        "operator@example.com",
				PasswordHash: string(hash),
				Status:       1,
			},
		},
		usersByID: map[int64]*model.User{
			9: {
				ID:           9,
				Username:     "operator",
				Email:        "operator@example.com",
				PasswordHash: string(hash),
				Status:       1,
			},
		},
		userRoles: map[int64][]string{
			9: []string{"operator"},
		},
	}
	if err := repo.EnsureRBACDefaults(context.Background()); err != nil {
		t.Fatalf("failed to seed fake rbac defaults: %v", err)
	}

	service := NewAuthService(repo, testAuthConfig())
	loginResp, err := service.Login(context.Background(), dto.LoginRequest{
		Username: "operator",
		Password: password,
	})
	if err != nil {
		t.Fatalf("expected login success, got %v", err)
	}
	claims, err := service.ParseToken(loginResp.AccessToken)
	if err != nil {
		t.Fatalf("expected access token parse success, got %v", err)
	}

	if err := service.RevokeSession(context.Background(), 9); err != nil {
		t.Fatalf("expected revoke success, got %v", err)
	}
	err = service.ValidateSession(context.Background(), claims)
	if err == nil {
		t.Fatalf("expected revoked session validation to fail")
	}
	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != apperror.ErrSessionExpired.Code {
		t.Fatalf("unexpected error code: %d", appErr.Code)
	}
}
