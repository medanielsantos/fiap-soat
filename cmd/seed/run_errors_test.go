package main

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestRunOnlyRunsLocally(t *testing.T) {
	for _, env := range []string{"", "production", "staging"} {
		t.Run("APP_ENV="+env, func(t *testing.T) {
			setSeedEnvironment(t, "postgres://unused/db")
			t.Setenv("APP_ENV", env)
			err := run(context.Background(), io.Discard)
			if err == nil || err.Error() != "seed only runs with APP_ENV=local" {
				t.Fatalf("err = %v, want local-only error", err)
			}
		})
	}
}

func TestRunRequiresDatabaseConfiguration(t *testing.T) {
	setSeedEnvironment(t, "")
	err := run(context.Background(), io.Discard)
	if err == nil || !strings.Contains(err.Error(), "database configuration is required") {
		t.Fatalf("err = %v, want database configuration error", err)
	}
}

func TestRunRequiresSeedPassword(t *testing.T) {
	setSeedEnvironment(t, "postgres://unused/db")
	t.Setenv("DEFAULT_PASSWORD_SEED", "")
	err := run(context.Background(), io.Discard)
	if err == nil || err.Error() != "DEFAULT_PASSWORD_SEED is required" {
		t.Fatalf("err = %v, want seed password error", err)
	}
}

func TestRunWrapsDatabaseOpenFailure(t *testing.T) {
	setSeedEnvironment(t, "postgres://nobody@127.0.0.1:1/none?sslmode=disable&connect_timeout=1")
	if err := run(context.Background(), io.Discard); err == nil || !strings.HasPrefix(err.Error(), "open database:") {
		t.Fatalf("err = %v, want open database error", err)
	}
}

func TestRunFailsWhenSchemaIsNotMigrated(t *testing.T) {
	dsn, _ := newSchemaDSN(t, false)
	setSeedEnvironment(t, dsn)
	if err := run(context.Background(), io.Discard); err == nil || !strings.HasPrefix(err.Error(), "seed roles and permissions:") {
		t.Fatalf("err = %v, want authorization seed error", err)
	}
}

func TestRunRejectsWeakSeedPassword(t *testing.T) {
	dsn, db := newSchemaDSN(t, true)
	setSeedEnvironment(t, dsn)
	t.Setenv("DEFAULT_PASSWORD_SEED", "short")

	if err := run(context.Background(), io.Discard); err == nil || !strings.HasPrefix(err.Error(), "seed user admin@oficina.local:") {
		t.Fatalf("err = %v, want seed user error", err)
	}
	var users int
	if err := db.QueryRow(`SELECT count(*) FROM users`).Scan(&users); err != nil || users != 0 {
		t.Fatalf("users = %d, err = %v; want none", users, err)
	}
}
