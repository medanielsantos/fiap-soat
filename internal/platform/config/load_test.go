package config

import (
	"testing"
	"time"
)

func TestLoadAppliesDefaultsWhenEnvironmentIsEmpty(t *testing.T) {
	clearConfigEnvironment(t)

	cfg := Load()
	if cfg.HTTPAddress != ":8080" {
		t.Errorf("HTTPAddress = %q, want :8080", cfg.HTTPAddress)
	}
	if cfg.DatabaseURL != "" {
		t.Errorf("DatabaseURL = %q, want empty without DB_NAME/DB_USER", cfg.DatabaseURL)
	}
	if cfg.JWTIssuer != "fiap-soat" || cfg.SMTPPort != "587" || cfg.JWTExpiry != 24*time.Hour {
		t.Errorf("unexpected defaults: issuer=%q smtpPort=%q expiry=%s", cfg.JWTIssuer, cfg.SMTPPort, cfg.JWTExpiry)
	}
}

func TestLoadReadsEveryVariable(t *testing.T) {
	clearConfigEnvironment(t)
	values := map[string]string{
		"APP_HTTP_PORT": "9090", "DATABASE_URL": "postgres://explicit/db", "REDIS_URL": "redis://cache:6379/0",
		"JWT_SECRET": "secret", "JWT_ISSUER": "issuer", "SMTP_HOST": "smtp.test", "SMTP_PORT": "2525",
		"SMTP_USERNAME": "user", "SMTP_PASSWORD": "pass", "SMTP_FROM": "from@test", "PASSWORD_RESET_URL": "https://reset",
		"CUSTOMER_PASSWORD_RESET_URL": "https://customer-reset", "ADMIN_NAME": "Admin", "ADMIN_EMAIL": "admin@test",
		"ADMIN_PASSWORD": "admin-pass", "DB_NAME": "ignored", "DB_USER": "ignored",
	}
	for key, value := range values {
		t.Setenv(key, value)
	}

	got := Load()
	want := Config{
		HTTPAddress: ":9090", DatabaseURL: "postgres://explicit/db", RedisURL: "redis://cache:6379/0",
		JWTSecret: "secret", JWTIssuer: "issuer", JWTExpiry: 24 * time.Hour, SMTPHost: "smtp.test", SMTPPort: "2525",
		SMTPUser: "user", SMTPPassword: "pass", SMTPFrom: "from@test", PasswordResetURL: "https://reset",
		CustomerPasswordResetURL: "https://customer-reset", AdminName: "Admin", AdminEmail: "admin@test", AdminPassword: "admin-pass",
	}
	if got != want {
		t.Fatalf("Load() = %+v\nwant %+v", got, want)
	}
}

func TestLoadBuildsDatabaseURLFromDBVariables(t *testing.T) {
	clearConfigEnvironment(t)
	t.Setenv("DB_NAME", "fiap")
	t.Setenv("DB_USER", "app")
	t.Setenv("DB_PASSWORD", "secret")

	got := Load().DatabaseURL
	if want := "postgres://app:secret@localhost:5432/fiap?sslmode=disable"; got != want {
		t.Fatalf("DatabaseURL = %q, want %q", got, want)
	}
}

func TestEnvOrFallsBackOnlyWhenUnsetOrEmpty(t *testing.T) {
	t.Setenv("FIAP_TEST_ENV_OR", "")
	if got := envOr("FIAP_TEST_ENV_OR", "fallback"); got != "fallback" {
		t.Fatalf("envOr(empty) = %q, want fallback", got)
	}
	t.Setenv("FIAP_TEST_ENV_OR", "value")
	if got := envOr("FIAP_TEST_ENV_OR", "fallback"); got != "value" {
		t.Fatalf("envOr(set) = %q, want value", got)
	}
}
