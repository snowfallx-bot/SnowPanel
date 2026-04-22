package service

import (
	"context"
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

func NewAuthService(userRepo repository.UserRepository, cfg config.AuthConfig) AuthService {
	return &authService{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

func (s *authService) EnsureDefaultAdmin(ctx context.Context) error {
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
	if count > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(s.cfg.DefaultAdminPassword), bcrypt.DefaultCost)
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

	token, expiresIn, err := s.generateToken(*user)
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
		User:        toProfile(*user),
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
	return toProfile(*user), nil
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

func (s *authService) generateToken(user model.User) (string, int64, error) {
	expireAt := time.Now().Add(s.cfg.JWTExpire)
	claims := &jwtClaims{
		UserID:      user.ID,
		Username:    user.Username,
		Roles:       defaultRolesForUser(user.Username),
		Permissions: defaultPermissionsForUser(user.Username),
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

func toProfile(user model.User) dto.UserProfile {
	return dto.UserProfile{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		Status:      user.Status,
		Roles:       defaultRolesForUser(user.Username),
		Permissions: defaultPermissionsForUser(user.Username),
	}
}

func defaultRolesForUser(username string) []string {
	if username == "admin" {
		return []string{"super_admin"}
	}
	return []string{"operator"}
}

func defaultPermissionsForUser(username string) []string {
	if username == "admin" {
		return []string{
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
	}
	return []string{
		"dashboard.read",
		"files.read",
	}
}

func (s *authService) emitAuditLogin(_ context.Context, _ string, _ bool) {
	// Reserved for audit integration in later stage.
}
