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
	createdUser     *model.User
	usersByName     map[string]*model.User
	usersByID       map[int64]*model.User
	userRoles       map[int64][]string
	rolePermissions map[string][]string
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
	if result.User.Username != "admin" {
		t.Fatalf("unexpected user profile username: %s", result.User.Username)
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
