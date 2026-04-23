package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"slices"
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
	Refresh(ctx context.Context, refreshToken string) (dto.LoginResponse, error)
	RevokeSession(ctx context.Context, userID int64) error
	ChangePassword(ctx context.Context, userID int64, req dto.ChangePasswordRequest) (dto.LoginResponse, error)
	Me(ctx context.Context, userID int64) (dto.UserProfile, error)
	ParseToken(token string) (TokenClaims, error)
	ValidateSession(ctx context.Context, claims TokenClaims) error
}

type TokenClaims struct {
	UserID             int64
	Username           string
	Roles              []string
	Permissions        []string
	MustChangePassword bool
	SessionIssuedAt    int64
	TokenUse           string
	RBACChecksum       string
}

type jwtClaims struct {
	UserID             int64    `json:"user_id"`
	Username           string   `json:"username"`
	Roles              []string `json:"roles"`
	Permissions        []string `json:"permissions"`
	MustChangePassword bool     `json:"must_change_password"`
	SessionIssuedAt    int64    `json:"session_issued_at"`
	TokenUse           string   `json:"token_use"`
	RBACChecksum       string   `json:"rbac_checksum"`
	jwt.RegisteredClaims
}

type authService struct {
	userRepo repository.UserRepository
	cfg      config.AuthConfig
}

const (
	roleSuperAdmin  = "super_admin"
	roleOperator    = "operator"
	tokenUseAccess  = "access"
	tokenUseRefresh = "refresh"
)

func NewAuthService(userRepo repository.UserRepository, cfg config.AuthConfig) AuthService {
	if cfg.JWTExpire <= 0 {
		cfg.JWTExpire = 24 * time.Hour
	}
	if cfg.JWTRefreshExpire <= 0 {
		cfg.JWTRefreshExpire = 7 * 24 * time.Hour
	}
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
		return dto.LoginResponse{}, apperror.ErrInvalidCredential
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		return dto.LoginResponse{}, apperror.ErrInvalidCredential
	}
	if user.Status != 1 {
		return dto.LoginResponse{}, apperror.ErrUserDisabled
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

	mustChangePassword := s.mustChangePassword(*user)
	sessionIssuedAt := nextSessionTime(user.LastLoginAt)
	if !mustChangePassword {
		if err := s.userRepo.UpdateLastLoginAt(ctx, user.ID, sessionIssuedAt); err != nil {
			return dto.LoginResponse{}, apperror.Wrap(
				apperror.ErrInternal.Code,
				apperror.ErrInternal.HTTPStatus,
				apperror.ErrInternal.Message,
				err,
			)
		}
		user.LastLoginAt = &sessionIssuedAt
	}

	accessToken, accessExpiresIn, refreshToken, refreshExpiresIn, err := s.generateTokenPair(
		*user,
		roles,
		permissions,
		mustChangePassword,
		sessionIssuedAt.UnixNano(),
	)
	if err != nil {
		return dto.LoginResponse{}, apperror.Wrap(
			apperror.ErrTokenGenerate.Code,
			apperror.ErrTokenGenerate.HTTPStatus,
			apperror.ErrTokenGenerate.Message,
			err,
		)
	}

	return dto.LoginResponse{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		TokenType:        "Bearer",
		ExpiresIn:        accessExpiresIn,
		RefreshExpiresIn: refreshExpiresIn,
		User:             toProfile(*user, roles, permissions, mustChangePassword),
	}, nil
}

func (s *authService) Refresh(ctx context.Context, refreshToken string) (dto.LoginResponse, error) {
	claims, err := s.parseTokenWithUse(refreshToken, tokenUseRefresh)
	if err != nil {
		return dto.LoginResponse{}, err
	}
	if claims.MustChangePassword {
		return dto.LoginResponse{}, apperror.ErrPasswordChangeNeed
	}
	if err := s.ValidateSession(ctx, claims); err != nil {
		return dto.LoginResponse{}, err
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return dto.LoginResponse{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if user == nil {
		return dto.LoginResponse{}, apperror.ErrSessionExpired
	}
	if user.Status != 1 {
		return dto.LoginResponse{}, apperror.ErrSessionExpired
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

	now := nextSessionTime(user.LastLoginAt)
	if err := s.userRepo.UpdateLastLoginAt(ctx, user.ID, now); err != nil {
		return dto.LoginResponse{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	user.LastLoginAt = &now

	accessToken, accessExpiresIn, nextRefreshToken, refreshExpiresIn, err := s.generateTokenPair(
		*user,
		roles,
		permissions,
		false,
		now.UnixNano(),
	)
	if err != nil {
		return dto.LoginResponse{}, apperror.Wrap(
			apperror.ErrTokenGenerate.Code,
			apperror.ErrTokenGenerate.HTTPStatus,
			apperror.ErrTokenGenerate.Message,
			err,
		)
	}

	return dto.LoginResponse{
		AccessToken:      accessToken,
		RefreshToken:     nextRefreshToken,
		TokenType:        "Bearer",
		ExpiresIn:        accessExpiresIn,
		RefreshExpiresIn: refreshExpiresIn,
		User:             toProfile(*user, roles, permissions, false),
	}, nil
}

func (s *authService) RevokeSession(ctx context.Context, userID int64) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if user == nil {
		return apperror.ErrUserNotFound
	}

	// Keep bootstrap admin password-rotation semantics stable:
	// if first-login flag is represented by nil last_login_at, do not clear it here.
	if s.mustChangePassword(*user) {
		return nil
	}

	now := nextSessionTime(user.LastLoginAt)
	if err := s.userRepo.UpdateLastLoginAt(ctx, user.ID, now); err != nil {
		return apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	return nil
}

func (s *authService) ChangePassword(
	ctx context.Context,
	userID int64,
	req dto.ChangePasswordRequest,
) (dto.LoginResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return dto.LoginResponse{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if user == nil {
		return dto.LoginResponse{}, apperror.ErrUserNotFound
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)) != nil {
		return dto.LoginResponse{}, apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			"current password is incorrect",
			errors.New("current password mismatch"),
		)
	}

	nextPassword := strings.TrimSpace(req.NewPassword)
	if !isStrongPassword(nextPassword) {
		return dto.LoginResponse{}, apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			"new password is too weak",
			errors.New("new password must be at least 14 chars and include upper/lower/digit/symbol"),
		)
	}

	if req.CurrentPassword == nextPassword {
		return dto.LoginResponse{}, apperror.Wrap(
			apperror.ErrBadRequest.Code,
			apperror.ErrBadRequest.HTTPStatus,
			"new password must be different from current password",
			errors.New("new password equals current password"),
		)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(nextPassword), bcrypt.DefaultCost)
	if err != nil {
		return dto.LoginResponse{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	if err := s.userRepo.UpdatePasswordHash(ctx, user.ID, string(hash)); err != nil {
		return dto.LoginResponse{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	now := nextSessionTime(user.LastLoginAt)
	if err := s.userRepo.UpdateLastLoginAt(ctx, user.ID, now); err != nil {
		return dto.LoginResponse{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	user.PasswordHash = string(hash)
	user.LastLoginAt = &now

	roles, permissions, err := s.resolveUserRBAC(ctx, *user)
	if err != nil {
		return dto.LoginResponse{}, apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}

	accessToken, accessExpiresIn, refreshToken, refreshExpiresIn, err := s.generateTokenPair(
		*user,
		roles,
		permissions,
		false,
		now.UnixNano(),
	)
	if err != nil {
		return dto.LoginResponse{}, apperror.Wrap(
			apperror.ErrTokenGenerate.Code,
			apperror.ErrTokenGenerate.HTTPStatus,
			apperror.ErrTokenGenerate.Message,
			err,
		)
	}

	return dto.LoginResponse{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		TokenType:        "Bearer",
		ExpiresIn:        accessExpiresIn,
		RefreshExpiresIn: refreshExpiresIn,
		User:             toProfile(*user, roles, permissions, false),
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
	return toProfile(*user, roles, permissions, s.mustChangePassword(*user)), nil
}

func (s *authService) ParseToken(rawToken string) (TokenClaims, error) {
	return s.parseTokenWithUse(rawToken, tokenUseAccess)
}

func (s *authService) parseTokenWithUse(rawToken string, expectedUse string) (TokenClaims, error) {
	claims := &jwtClaims{}
	parsed, err := jwt.ParseWithClaims(rawToken, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !parsed.Valid {
		return TokenClaims{}, apperror.ErrTokenParse
	}

	tokenUse := strings.TrimSpace(strings.ToLower(claims.TokenUse))
	if tokenUse == "" {
		tokenUse = tokenUseAccess
	}
	if expectedUse != "" && tokenUse != expectedUse {
		return TokenClaims{}, apperror.ErrTokenParse
	}

	return TokenClaims{
		UserID:             claims.UserID,
		Username:           claims.Username,
		Roles:              claims.Roles,
		Permissions:        claims.Permissions,
		MustChangePassword: claims.MustChangePassword,
		SessionIssuedAt:    claims.SessionIssuedAt,
		TokenUse:           tokenUse,
		RBACChecksum:       strings.TrimSpace(claims.RBACChecksum),
	}, nil
}

func (s *authService) ValidateSession(ctx context.Context, claims TokenClaims) error {
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	if user == nil {
		return apperror.ErrSessionExpired
	}
	if user.Status != 1 {
		return apperror.ErrSessionExpired
	}
	if claims.SessionIssuedAt <= 0 {
		return apperror.ErrTokenParse
	}
	if user.LastLoginAt != nil && claims.SessionIssuedAt < user.LastLoginAt.UnixNano() {
		return apperror.ErrSessionExpired
	}

	currentRoles, currentPermissions, err := s.resolveUserRBAC(ctx, *user)
	if err != nil {
		return apperror.Wrap(
			apperror.ErrInternal.Code,
			apperror.ErrInternal.HTTPStatus,
			apperror.ErrInternal.Message,
			err,
		)
	}
	currentChecksum := computeRBACChecksum(currentRoles, currentPermissions)
	claimChecksum := strings.TrimSpace(claims.RBACChecksum)
	if claimChecksum == "" {
		claimChecksum = computeRBACChecksum(claims.Roles, claims.Permissions)
	}
	if claimChecksum != currentChecksum {
		return apperror.ErrSessionExpired
	}

	return nil
}

func (s *authService) generateToken(
	user model.User,
	roles []string,
	permissions []string,
	mustChangePassword bool,
	tokenUse string,
	rbacChecksum string,
	expireAfter time.Duration,
	sessionIssuedAt int64,
) (string, int64, error) {
	issuedAt := time.Now().UTC()
	expireAt := issuedAt.Add(expireAfter)
	tokenID, err := generateTokenID(16)
	if err != nil {
		return "", 0, err
	}
	claims := &jwtClaims{
		UserID:             user.ID,
		Username:           user.Username,
		Roles:              roles,
		Permissions:        permissions,
		MustChangePassword: mustChangePassword,
		SessionIssuedAt:    sessionIssuedAt,
		TokenUse:           tokenUse,
		RBACChecksum:       rbacChecksum,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireAt),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			Issuer:    s.cfg.JWTIssuer,
			Subject:   user.Username,
			ID:        tokenID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", 0, err
	}
	return signed, int64(expireAfter.Seconds()), nil
}

func (s *authService) generateTokenPair(
	user model.User,
	roles []string,
	permissions []string,
	mustChangePassword bool,
	sessionIssuedAt int64,
) (string, int64, string, int64, error) {
	if sessionIssuedAt <= 0 {
		sessionIssuedAt = time.Now().UTC().UnixNano()
	}
	rbacChecksum := computeRBACChecksum(roles, permissions)

	accessToken, accessExpiresIn, err := s.generateToken(
		user,
		roles,
		permissions,
		mustChangePassword,
		tokenUseAccess,
		rbacChecksum,
		s.cfg.JWTExpire,
		sessionIssuedAt,
	)
	if err != nil {
		return "", 0, "", 0, err
	}

	refreshToken, refreshExpiresIn, err := s.generateToken(
		user,
		roles,
		permissions,
		mustChangePassword,
		tokenUseRefresh,
		rbacChecksum,
		s.cfg.JWTRefreshExpire,
		sessionIssuedAt,
	)
	if err != nil {
		return "", 0, "", 0, err
	}

	return accessToken, accessExpiresIn, refreshToken, refreshExpiresIn, nil
}

func nextSessionTime(previous *time.Time) time.Time {
	now := time.Now().UTC()
	if previous == nil {
		return now
	}
	if !now.After(*previous) {
		return previous.Add(time.Nanosecond)
	}
	return now
}

func computeRBACChecksum(roles []string, permissions []string) string {
	normalizedRoles := append([]string(nil), roles...)
	normalizedPermissions := append([]string(nil), permissions...)
	slices.Sort(normalizedRoles)
	slices.Sort(normalizedPermissions)

	payload := strings.Join(normalizedRoles, ",") + "|" + strings.Join(normalizedPermissions, ",")
	digest := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(digest[:])
}

func toProfile(
	user model.User,
	roles []string,
	permissions []string,
	mustChangePassword bool,
) dto.UserProfile {
	return dto.UserProfile{
		ID:                 user.ID,
		Username:           user.Username,
		Email:              user.Email,
		Status:             user.Status,
		Roles:              roles,
		Permissions:        permissions,
		MustChangePassword: mustChangePassword,
	}
}

func (s *authService) mustChangePassword(user model.User) bool {
	if !strings.EqualFold(strings.TrimSpace(user.Username), strings.TrimSpace(s.cfg.DefaultAdminUsername)) {
		return false
	}
	return user.LastLoginAt == nil
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

func generateTokenID(byteLen int) (string, error) {
	buffer := make([]byte, byteLen)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
