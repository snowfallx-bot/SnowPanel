package config

import (
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
	JWTSecret            string
	JWTIssuer            string
	JWTExpire            time.Duration
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
	v.SetDefault("JWT_SECRET", "change-me-in-production")
	v.SetDefault("JWT_ISSUER", "snowpanel-backend")
	v.SetDefault("JWT_EXPIRE", "24h")
	v.SetDefault("BOOTSTRAP_ADMIN", true)
	v.SetDefault("DEFAULT_ADMIN_USERNAME", "admin")
	v.SetDefault("DEFAULT_ADMIN_EMAIL", "admin@snowpanel.local")
	v.SetDefault("DEFAULT_ADMIN_PASSWORD", "admin123456")

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

	return Config{
		AppEnv:      v.GetString("APP_ENV"),
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
			JWTSecret:            v.GetString("JWT_SECRET"),
			JWTIssuer:            v.GetString("JWT_ISSUER"),
			JWTExpire:            mustDuration(v.GetString("JWT_EXPIRE"), 24*time.Hour),
			BootstrapAdmin:       v.GetBool("BOOTSTRAP_ADMIN"),
			DefaultAdminUsername: v.GetString("DEFAULT_ADMIN_USERNAME"),
			DefaultAdminEmail:    v.GetString("DEFAULT_ADMIN_EMAIL"),
			DefaultAdminPassword: v.GetString("DEFAULT_ADMIN_PASSWORD"),
		},
	}
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
