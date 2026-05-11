package config

import "testing"

func TestValidateRejectsWeakJWTSecretInProduction(t *testing.T) {
	cfg := Config{
		AppEnv: "production",
		Auth: AuthConfig{
			AppEnv:               "production",
			JWTSecret:            "change-me-in-production",
			BootstrapAdmin:       false,
			DefaultAdminUsername: "admin",
			DefaultAdminEmail:    "admin@example.com",
			DefaultAdminPassword: "",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error for weak JWT secret in production")
	}
}

func TestValidateRejectsWeakBootstrapPasswordInProduction(t *testing.T) {
	cfg := Config{
		AppEnv: "production",
		Auth: AuthConfig{
			AppEnv:               "production",
			JWTSecret:            "VeryStrongJWTSecret_For_Production_Use_1234567890!",
			BootstrapAdmin:       true,
			DefaultAdminUsername: "admin",
			DefaultAdminEmail:    "admin@example.com",
			DefaultAdminPassword: "admin123456",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error for weak bootstrap password in production")
	}
}

func TestValidateAllowsStrongProductionConfig(t *testing.T) {
	cfg := Config{
		AppEnv: "production",
		Auth: AuthConfig{
			AppEnv:               "production",
			JWTSecret:            "VeryStrongJWTSecret_For_Production_Use_1234567890!",
			BootstrapAdmin:       true,
			DefaultAdminUsername: "admin",
			DefaultAdminEmail:    "admin@example.com",
			DefaultAdminPassword: "Str0ng!BootstrapP@ssword",
		},
		TaskWorker: TaskWorkerConfig{
			Concurrency:   2,
			LeaseDuration: 30,
			PollInterval:  2,
			MaxAttempts:   3,
		},
		Audit: AuditConfig{
			RetentionDays: 180,
			ExportMaxRows: 100000,
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected config to be valid, got %v", err)
	}
}

func TestValidateRejectsMissingBackendAgentToken(t *testing.T) {
	cfg := Config{
		AppEnv: "production",
		Auth: AuthConfig{
			AppEnv:               "production",
			JWTSecret:            "VeryStrongJWTSecret_For_Production_Use_1234567890!",
			BootstrapAdmin:       false,
			DefaultAdminUsername: "admin",
			DefaultAdminEmail:    "admin@example.com",
		},
		AgentAuth: AgentAuthConfig{
			Mode: "token",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error for missing backend agent token")
	}
}

func TestValidateRejectsUnsupportedBackendAgentMTLS(t *testing.T) {
	cfg := Config{
		AppEnv: "production",
		Auth: AuthConfig{
			AppEnv:               "production",
			JWTSecret:            "VeryStrongJWTSecret_For_Production_Use_1234567890!",
			BootstrapAdmin:       false,
			DefaultAdminUsername: "admin",
			DefaultAdminEmail:    "admin@example.com",
		},
		AgentAuth: AgentAuthConfig{
			Mode: "mtls",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error for unsupported backend agent mtls mode")
	}
}

func TestValidateRejectsUnknownLoginAttemptStore(t *testing.T) {
	cfg := Config{
		AppEnv: "production",
		Auth: AuthConfig{
			AppEnv:               "production",
			JWTSecret:            "VeryStrongJWTSecret_For_Production_Use_1234567890!",
			BootstrapAdmin:       false,
			DefaultAdminUsername: "admin",
			DefaultAdminEmail:    "admin@example.com",
			LoginAttemptStore:    "etcd",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error for unknown LOGIN_ATTEMPT_STORE")
	}
}

func TestValidateRejectsInvalidEncryptionKey(t *testing.T) {
	cfg := Config{
		AppEnv: "production",
		Auth: AuthConfig{
			AppEnv:               "production",
			JWTSecret:            "VeryStrongJWTSecret_For_Production_Use_1234567890!",
			BootstrapAdmin:       false,
			DefaultAdminUsername: "admin",
			DefaultAdminEmail:    "admin@example.com",
		},
		Security: SecurityConfig{
			EncryptionKey: "too-short",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error for invalid encryption key")
	}
}

func TestValidateRejectsInvalidTaskWorkerConfig(t *testing.T) {
	base := Config{
		AppEnv: "production",
		Auth: AuthConfig{
			AppEnv:               "production",
			JWTSecret:            "VeryStrongJWTSecret_For_Production_Use_1234567890!",
			BootstrapAdmin:       false,
			DefaultAdminUsername: "admin",
			DefaultAdminEmail:    "admin@example.com",
		},
		TaskWorker: TaskWorkerConfig{
			Concurrency:   2,
			LeaseDuration: 30,
			PollInterval:  2,
			MaxAttempts:   3,
		},
	}

	cases := []struct {
		name   string
		mutate func(*Config)
	}{
		{
			name: "zero concurrency",
			mutate: func(cfg *Config) {
				cfg.TaskWorker.Concurrency = 0
			},
		},
		{
			name: "zero lease duration",
			mutate: func(cfg *Config) {
				cfg.TaskWorker.LeaseDuration = 0
			},
		},
		{
			name: "zero poll interval",
			mutate: func(cfg *Config) {
				cfg.TaskWorker.PollInterval = 0
			},
		},
		{
			name: "zero max attempts",
			mutate: func(cfg *Config) {
				cfg.TaskWorker.MaxAttempts = 0
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base
			tc.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatalf("expected validation error")
			}
		})
	}
}

func TestValidateRejectsInvalidAuditConfig(t *testing.T) {
	base := Config{
		AppEnv: "production",
		Auth: AuthConfig{
			AppEnv:               "production",
			JWTSecret:            "VeryStrongJWTSecret_For_Production_Use_1234567890!",
			BootstrapAdmin:       false,
			DefaultAdminUsername: "admin",
			DefaultAdminEmail:    "admin@example.com",
		},
		TaskWorker: TaskWorkerConfig{
			Concurrency:   2,
			LeaseDuration: 30,
			PollInterval:  2,
			MaxAttempts:   3,
		},
		Audit: AuditConfig{
			RetentionDays: 180,
			ExportMaxRows: 100000,
		},
	}

	cases := []struct {
		name   string
		mutate func(*Config)
	}{
		{
			name: "zero retention days",
			mutate: func(cfg *Config) {
				cfg.Audit.RetentionDays = 0
			},
		},
		{
			name: "zero export max rows",
			mutate: func(cfg *Config) {
				cfg.Audit.ExportMaxRows = 0
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base
			tc.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatalf("expected validation error")
			}
		})
	}
}
