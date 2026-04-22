package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/apperror"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/config"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/dto"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/model"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	EnsureDefaultAdmin(ctx context.Context) error
	Login(ctx context.Context, req dto.LoginRequest) (dto.LoginResponse, error)
	Me(ctx context.Context, userID int64) (dto.UserProfile, error)
	ParseToken(token string) (TokenClaims, error)
}

type TokenClaims struct {
	UserID      int64
	Username    string
	Roles       []string
	Permissions []string
}

type jwtClaims struct {
	UserID      int64    `json:"user_id"`
	Username    string   `json:"username"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

type authService struct {
	userRepo repository.UserRepository
	cfg      config.AuthConfig
}

const (
	roleSuperAdmin = "super_admin"
	roleOperator   = "operator"
)

func NewAuthService(userRepo repository.UserRepository, cfg config.AuthConfig) AuthService {
	return &authService{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

func (s *authService) EnsureDefaultAdmin(ctx context.Context) error {
	if err := s.userRepo.EnsureRBACDefaults(ctx); err != nil {
		return apperror.Wrap(
			apperror.ErrBootstrapAdminFail.Code,
			apperror.ErrBootstrapAdminFail.HTTPStatus,
			apperror.ErrBootstrapAdminFail.Message,
			err,
		)
	}

	if !s.cfg.BootstrapAdmin {
		return nil
	}

	count, err := s.userRepo.Count(ctx)
	if err != nil {
		return apperror.Wrap(
			apperror.ErrBootstrapAdminFail.Code,
			apperror.ErrBootstrapAdminFail.HTTPStatus,
			apperror.ErrBootstrapAdminFail.Message,
			err,
		)
	}
	var bootstrapUser *model.User
	if count == 0 {
		bootstrapPassword, err := s.resolveBootstrapPassword()
		if err != nil {
			return apperror.Wrap(
				apperror.ErrBootstrapAdminFail.Code,
				apperror.ErrBootstrapAdminFail.HTTPStatus,
				apperror.ErrBootstrapAdminFail.Message,
				err,
			)
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(bootstrapPassword), bcrypt.DefaultCost)
		if err != nil {
			return apperror.Wrap(
				apperror.ErrBootstrapAdminFail.Code,
				apperror.ErrBootstrapAdminFail.HTTPStatus,
				apperror.ErrBootstrapAdminFail.Message,
				err,
			)
		}

		user := &model.User{
			Username:     s.cfg.DefaultAdminUsername,
			Email:        s.cfg.DefaultAdminEmail,
			PasswordHash: string(hash),
			Status:       1,
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			return apperror.Wrap(
				apperror.ErrBootstrapAdminFail.Code,
				apperror.ErrBootstrapAdminFail.HTTPStatus,
				apperror.ErrBootstrapAdminFail.Message,
				err,
			)
		}
		bootstrapUser = user
	}

	if bootstrapUser == nil {
		existing, err := s.userRepo.GetByUsername(ctx, s.cfg.DefaultAdminUsername)
		if err != nil {
			return apperror.Wrap(
				apperror.ErrBootstrapAdminFail.Code,
				apperror.ErrBootstrapAdminFail.HTTPStatus,
				apperror.ErrBootstrapAdminFail.Message,
				err,
			)
		}
		bootstrapUser = existing
	}

	if bootstrapUser != nil {
		if err := s.userRepo.EnsureUserRoleBySlug(ctx, bootstrapUser.ID, roleSuperAdmin); err != nil {
			return apperror.Wrap(
				apperror.ErrBootstrapAdminFail.Code,
				apperror.ErrBootstrapAdminFail.HTTPStatus,
				apperror.ErrBootstrapAdminFail.Message,
				err,
			)
		}
	}
	return nil
}

func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (dto.LoginResponse, error) {
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		return dto.LoginResponse{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if user == nil {
		s.emitAuditLogin(ctx, req.Username, false)
		return dto.LoginResponse{}, apperror.ErrInvalidCredential
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		s.emitAuditLogin(ctx, user.Username, false)
		return dto.LoginResponse{}, apperror.ErrInvalidCredential
	}

	roles, permissions, err := s.resolveUserRBAC(ctx, *user)
	if err != nil {
		return dto.LoginResponse{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	token, expiresIn, err := s.generateToken(*user, roles, permissions)
	if err != nil {
		return dto.LoginResponse{}, apperror.Wrap(
			apperror.ErrTokenGenerate.Code,
			apperror.ErrTokenGenerate.HTTPStatus,
			apperror.ErrTokenGenerate.Message,
			err,
		)
	}

	s.emitAuditLogin(ctx, user.Username, true)
	return dto.LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		User:        toProfile(*user, roles, permissions),
	}, nil
}

func (s *authService) Me(ctx context.Context, userID int64) (dto.UserProfile, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return dto.UserProfile{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if user == nil {
		return dto.UserProfile{}, apperror.ErrUserNotFound
	}

	roles, permissions, err := s.resolveUserRBAC(ctx, *user)
	if err != nil {
		return dto.UserProfile{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	return toProfile(*user, roles, permissions), nil
}

func (s *authService) ParseToken(rawToken string) (TokenClaims, error) {
	claims := &jwtClaims{}
	parsed, err := jwt.ParseWithClaims(rawToken, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !parsed.Valid {
		return TokenClaims{}, apperror.ErrTokenParse
	}
	return TokenClaims{
		UserID:      claims.UserID,
		Username:    claims.Username,
		Roles:       claims.Roles,
		Permissions: claims.Permissions,
	}, nil
}

func (s *authService) generateToken(user model.User, roles []string, permissions []string) (string, int64, error) {
	expireAt := time.Now().Add(s.cfg.JWTExpire)
	claims := &jwtClaims{
		UserID:      user.ID,
		Username:    user.Username,
		Roles:       roles,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    s.cfg.JWTIssuer,
			Subject:   user.Username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", 0, err
	}
	return signed, int64(s.cfg.JWTExpire.Seconds()), nil
}

func toProfile(user model.User, roles []string, permissions []string) dto.UserProfile {
	return dto.UserProfile{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		Status:      user.Status,
		Roles:       roles,
		Permissions: permissions,
	}
}

func (s *authService) emitAuditLogin(_ context.Context, _ string, _ bool) {
	// Reserved for audit integration in later stage.
}

func (s *authService) resolveUserRBAC(ctx context.Context, user model.User) ([]string, []string, error) {
	roles, permissions, err := s.userRepo.GetRolesAndPermissions(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}
	if len(roles) > 0 {
		return roles, permissions, nil
	}

	if err := s.userRepo.EnsureUserRoleBySlug(ctx, user.ID, roleOperator); err != nil {
		return nil, nil, err
	}

	roles, permissions, err = s.userRepo.GetRolesAndPermissions(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}
	return roles, permissions, nil
}

func (s *authService) resolveBootstrapPassword() (string, error) {
	username := strings.TrimSpace(s.cfg.DefaultAdminUsername)
	email := strings.TrimSpace(s.cfg.DefaultAdminEmail)
	if username == "" {
		return "", errors.New("DEFAULT_ADMIN_USERNAME cannot be empty")
	}
	if email == "" {
		return "", errors.New("DEFAULT_ADMIN_EMAIL cannot be empty")
	}

	password := strings.TrimSpace(s.cfg.DefaultAdminPassword)
	if password == "" {
		if isProductionEnv(s.cfg.AppEnv) {
			return "", errors.New("DEFAULT_ADMIN_PASSWORD must be set in production")
		}

		generated, err := generateBootstrapPassword(24)
		if err != nil {
			return "", fmt.Errorf("generate bootstrap password: %w", err)
		}
		log.Printf(
			"[security] generated bootstrap admin password for user '%s': %s",
			username,
			generated,
		)
		return generated, nil
	}

	if isProductionEnv(s.cfg.AppEnv) && !isStrongPassword(password) {
		return "", errors.New("DEFAULT_ADMIN_PASSWORD is weak for production environment")
	}

	if !isStrongPassword(password) {
		log.Printf(
			"[security] warning: bootstrap admin password for user '%s' is weak; rotate it after initial login",
			username,
		)
	}

	return password, nil
}

func isProductionEnv(raw string) bool {
	return strings.EqualFold(strings.TrimSpace(raw), "production")
}

func isStrongPassword(raw string) bool {
	password := strings.TrimSpace(raw)
	if len(password) < 14 {
		return false
	}

	var hasUpper bool
	var hasLower bool
	var hasDigit bool
	var hasSymbol bool

	for _, ch := range password {
		switch {
		case ch >= 'A' && ch <= 'Z':
			hasUpper = true
		case ch >= 'a' && ch <= 'z':
			hasLower = true
		case ch >= '0' && ch <= '9':
			hasDigit = true
		default:
			hasSymbol = true
		}
	}

	return hasUpper && hasLower && hasDigit && hasSymbol
}

func generateBootstrapPassword(byteLen int) (string, error) {
	buffer := make([]byte, byteLen)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
