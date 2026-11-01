package config

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvFilePreservesExistingEnvironment(t *testing.T) {
	const fileKey = "FIAP_TEST_DOTENV_FILE_VALUE"
	const existingKey = "FIAP_TEST_DOTENV_EXISTING_VALUE"
	oldValue, existed := os.LookupEnv(fileKey)
	_ = os.Unsetenv(fileKey)
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(fileKey, oldValue)
		} else {
			_ = os.Unsetenv(fileKey)
		}
	})
	t.Setenv(existingKey, "from-environment")

	path := filepath.Join(t.TempDir(), "env.testing")
	contents := fileKey + `="from-file"` + "\n" + existingKey + `="should-not-win"` + "\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	LoadDotEnvFile(path)
	if got := os.Getenv(fileKey); got != "from-file" {
		t.Fatalf("file value = %q, want from-file", got)
	}
	if got := os.Getenv(existingKey); got != "from-environment" {
		t.Fatalf("existing environment value = %q, want from-environment", got)
	}
}

func TestPostgresURLFromEnvironmentUsesTestDBVariables(t *testing.T) {
	t.Setenv("DB_HOST", "db.testing")
	t.Setenv("DB_PORT", "5544")
	t.Setenv("DB_NAME", "fiap_test")
	t.Setenv("DB_USER", "test-user")
	t.Setenv("DB_PASSWORD", "p@ss word")
	t.Setenv("DB_SSL_MODE", "require")
	t.Setenv("DATABASE_URL", "postgres://production.invalid/ignored")

	parsed, err := url.Parse(PostgresURLFromEnvironment())
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Host != "db.testing:5544" || parsed.Path != "/fiap_test" || parsed.User.Username() != "test-user" {
		t.Fatalf("unexpected test database URL: %s", parsed.Redacted())
	}
	password, _ := parsed.User.Password()
	if password != "p@ss word" || parsed.Query().Get("sslmode") != "require" {
		t.Fatalf("test URL lost credentials or SSL mode: %s", parsed.Redacted())
	}
}
