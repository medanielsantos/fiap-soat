package config

import (
	"os"
	"testing"
)

func clearConfigEnvironment(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"APP_HTTP_PORT", "DATABASE_URL", "DB_HOST", "DB_PORT", "DB_NAME", "DB_USER", "DB_PASSWORD", "DB_SSL_MODE",
		"REDIS_URL", "JWT_SECRET", "JWT_ISSUER", "SMTP_HOST", "SMTP_PORT", "SMTP_USERNAME", "SMTP_PASSWORD", "SMTP_FROM",
		"PASSWORD_RESET_URL", "CUSTOMER_PASSWORD_RESET_URL", "ADMIN_NAME", "ADMIN_EMAIL", "ADMIN_PASSWORD",
	} {
		t.Setenv(key, "")
	}
}

func unsetForTest(t *testing.T, key string) {
	t.Helper()
	old, existed := os.LookupEnv(key)
	_ = os.Unsetenv(key)
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(key, old)
			return
		}
		_ = os.Unsetenv(key)
	})
}
