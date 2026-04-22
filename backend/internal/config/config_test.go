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
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected config to be valid, got %v", err)
	}
}
