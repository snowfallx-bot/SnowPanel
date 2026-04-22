package config

import (
	"os"
	"time"
)

const (
	defaultHost         = "0.0.0.0"
	defaultPort         = "8080"
	defaultReadTimeout  = 5 * time.Second
	defaultWriteTimeout = 10 * time.Second
)

type Config struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func Load() Config {
	return Config{
		Host:         readEnv("BACKEND_HOST", defaultHost),
		Port:         readEnv("BACKEND_PORT", defaultPort),
		ReadTimeout:  readEnvDuration("BACKEND_READ_TIMEOUT", defaultReadTimeout),
		WriteTimeout: readEnvDuration("BACKEND_WRITE_TIMEOUT", defaultWriteTimeout),
	}
}

func (c Config) Address() string {
	return c.Host + ":" + c.Port
}

func readEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func readEnvDuration(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}
