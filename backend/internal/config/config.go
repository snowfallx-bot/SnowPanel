package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	AppEnv       string
	Server       ServerConfig
	Database     DatabaseConfig
	Auth         AuthConfig
	AgentTarget  string
	AgentTimeout time.Duration
}

type ServerConfig struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	Host         string
	Port         int
	User         string
	Password     string
	Name         string
	SSLMode      string
	Timezone     string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  time.Duration
}

type AuthConfig struct {
	AppEnv               string
	JWTSecret            string
	JWTIssuer            string
	JWTExpire            time.Duration
	LoginMaxFailures     int
	LoginFailureWindow   time.Duration
	LoginLockDuration    time.Duration
	BootstrapAdmin       bool
	DefaultAdminUsername string
	DefaultAdminEmail    string
	DefaultAdminPassword string
}

func Load() Config {
	v := viper.New()
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	v.AddConfigPath("..")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	_ = v.ReadInConfig()

	v.SetDefault("APP_ENV", "development")
	v.SetDefault("BACKEND_HOST", "0.0.0.0")
	v.SetDefault("BACKEND_PORT", "8080")
	v.SetDefault("BACKEND_READ_TIMEOUT", "5s")
	v.SetDefault("BACKEND_WRITE_TIMEOUT", "10s")
	v.SetDefault("AGENT_TARGET", "127.0.0.1:50051")
	v.SetDefault("AGENT_TIMEOUT", "3s")
	v.SetDefault("JWT_SECRET", "")
	v.SetDefault("JWT_ISSUER", "snowpanel-backend")
	v.SetDefault("JWT_EXPIRE", "24h")
	v.SetDefault("LOGIN_MAX_FAILURES", 5)
	v.SetDefault("LOGIN_FAILURE_WINDOW", "15m")
	v.SetDefault("LOGIN_LOCK_DURATION", "15m")
	v.SetDefault("BOOTSTRAP_ADMIN", true)
	v.SetDefault("DEFAULT_ADMIN_USERNAME", "admin")
	v.SetDefault("DEFAULT_ADMIN_EMAIL", "admin@snowpanel.local")
	v.SetDefault("DEFAULT_ADMIN_PASSWORD", "")

	v.SetDefault("POSTGRES_HOST", "127.0.0.1")
	v.SetDefault("POSTGRES_PORT", 5432)
	v.SetDefault("POSTGRES_USER", "snowpanel")
	v.SetDefault("POSTGRES_PASSWORD", "snowpanel")
	v.SetDefault("POSTGRES_DB", "snowpanel")
	v.SetDefault("POSTGRES_SSLMODE", "disable")
	v.SetDefault("POSTGRES_TIMEZONE", "UTC")
	v.SetDefault("POSTGRES_MAX_OPEN_CONNS", 20)
	v.SetDefault("POSTGRES_MAX_IDLE_CONNS", 5)
	v.SetDefault("POSTGRES_CONN_MAX_LIFETIME", "30m")

	appEnv := normalizeEnv(v.GetString("APP_ENV"))
	jwtSecret := strings.TrimSpace(v.GetString("JWT_SECRET"))
	if !isProductionEnv(appEnv) && isWeakJWTSecret(jwtSecret) {
		generated, err := generateDevJWTSecret(48)
		if err == nil {
			jwtSecret = generated
		}
	}

	return Config{
		AppEnv:      appEnv,
		AgentTarget: v.GetString("AGENT_TARGET"),
		AgentTimeout: mustDuration(
			v.GetString("AGENT_TIMEOUT"),
			3*time.Second,
		),
		Server: ServerConfig{
			Host:         v.GetString("BACKEND_HOST"),
			Port:         v.GetString("BACKEND_PORT"),
			ReadTimeout:  mustDuration(v.GetString("BACKEND_READ_TIMEOUT"), 5*time.Second),
			WriteTimeout: mustDuration(v.GetString("BACKEND_WRITE_TIMEOUT"), 10*time.Second),
		},
		Database: DatabaseConfig{
			Host:         v.GetString("POSTGRES_HOST"),
			Port:         v.GetInt("POSTGRES_PORT"),
			User:         v.GetString("POSTGRES_USER"),
			Password:     v.GetString("POSTGRES_PASSWORD"),
			Name:         v.GetString("POSTGRES_DB"),
			SSLMode:      v.GetString("POSTGRES_SSLMODE"),
			Timezone:     v.GetString("POSTGRES_TIMEZONE"),
			MaxOpenConns: v.GetInt("POSTGRES_MAX_OPEN_CONNS"),
			MaxIdleConns: v.GetInt("POSTGRES_MAX_IDLE_CONNS"),
			MaxLifetime: mustDuration(
				v.GetString("POSTGRES_CONN_MAX_LIFETIME"),
				30*time.Minute,
			),
		},
		Auth: AuthConfig{
			AppEnv:               appEnv,
			JWTSecret:            jwtSecret,
			JWTIssuer:            v.GetString("JWT_ISSUER"),
			JWTExpire:            mustDuration(v.GetString("JWT_EXPIRE"), 24*time.Hour),
			LoginMaxFailures:     v.GetInt("LOGIN_MAX_FAILURES"),
			LoginFailureWindow:   mustDuration(v.GetString("LOGIN_FAILURE_WINDOW"), 15*time.Minute),
			LoginLockDuration:    mustDuration(v.GetString("LOGIN_LOCK_DURATION"), 15*time.Minute),
			BootstrapAdmin:       v.GetBool("BOOTSTRAP_ADMIN"),
			DefaultAdminUsername: v.GetString("DEFAULT_ADMIN_USERNAME"),
			DefaultAdminEmail:    v.GetString("DEFAULT_ADMIN_EMAIL"),
			DefaultAdminPassword: v.GetString("DEFAULT_ADMIN_PASSWORD"),
		},
	}
}

func (c Config) Validate() error {
	if isProductionEnv(c.AppEnv) {
		if isWeakJWTSecret(c.Auth.JWTSecret) {
			return errors.New("JWT_SECRET must be explicitly set to a strong value in production")
		}
	}

	if c.Auth.BootstrapAdmin {
		if strings.TrimSpace(c.Auth.DefaultAdminUsername) == "" {
			return errors.New("DEFAULT_ADMIN_USERNAME cannot be empty when BOOTSTRAP_ADMIN=true")
		}
		if strings.TrimSpace(c.Auth.DefaultAdminEmail) == "" {
			return errors.New("DEFAULT_ADMIN_EMAIL cannot be empty when BOOTSTRAP_ADMIN=true")
		}
		if isProductionEnv(c.AppEnv) && !isStrongPassword(c.Auth.DefaultAdminPassword) {
			return errors.New("DEFAULT_ADMIN_PASSWORD must be strong in production when BOOTSTRAP_ADMIN=true")
		}
	}

	return nil
}

func (c ServerConfig) Address() string {
	return c.Host + ":" + c.Port
}

func mustDuration(raw string, fallback time.Duration) time.Duration {
	if raw == "" {
		return fallback
	}

	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}

func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		c.Host,
		c.Port,
		c.User,
		c.Password,
		c.Name,
		c.SSLMode,
		c.Timezone,
	)
}

func normalizeEnv(raw string) string {
	normalized := strings.TrimSpace(strings.ToLower(raw))
	if normalized == "" {
		return "development"
	}
	return normalized
}

func isProductionEnv(raw string) bool {
	return normalizeEnv(raw) == "production"
}

func isWeakJWTSecret(secret string) bool {
	trimmed := strings.TrimSpace(secret)
	if len(trimmed) < 32 {
		return true
	}

	lowered := strings.ToLower(trimmed)
	weakValues := []string{
		"change-me-in-production",
		"changeme",
		"password",
		"admin123456",
		"snowpanel",
	}
	for _, weak := range weakValues {
		if strings.Contains(lowered, weak) {
			return true
		}
	}
	return false
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

func generateDevJWTSecret(byteLen int) (string, error) {
	buffer := make([]byte, byteLen)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
