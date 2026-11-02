package containers

import (
	"os"
	"testing"
)

const testJWTSecret = "0123456789abcdef0123456789abcdef"

func setValidEnvironment(t *testing.T) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to resolve the dependency container")
	}
	for key, value := range map[string]string{
		"DATABASE_URL": dsn, "DB_NAME": "", "DB_USER": "", "REDIS_URL": "redis://127.0.0.1:1/0",
		"JWT_SECRET": testJWTSecret, "JWT_ISSUER": "", "APP_HTTP_PORT": "", "SMTP_HOST": "", "SMTP_FROM": "",
	} {
		t.Setenv(key, value)
	}
}
