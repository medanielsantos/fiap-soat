package main

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestRunRequiresDatabaseConfiguration(t *testing.T) {
	setSeedEnvironment(t, "")
	err := run(context.Background(), io.Discard)
	if err == nil || !strings.Contains(err.Error(), "database configuration is required") {
		t.Fatalf("err = %v, want database configuration error", err)
	}
}

func TestRunRequiresAdministratorCredentials(t *testing.T) {
	for _, key := range []string{"ADMIN_NAME", "ADMIN_EMAIL", "ADMIN_PASSWORD"} {
		t.Run(key, func(t *testing.T) {
			setSeedEnvironment(t, "postgres://unused/db")
			t.Setenv(key, "")
			err := run(context.Background(), io.Discard)
			if err == nil || err.Error() != "ADMIN_NAME, ADMIN_EMAIL and ADMIN_PASSWORD are required" {
				t.Fatalf("err = %v, want admin credentials error", err)
			}
		})
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

func TestRunRejectsInvalidAdministratorDetails(t *testing.T) {
	dsn, db := newSchemaDSN(t, true)
	setSeedEnvironment(t, dsn)
	t.Setenv("ADMIN_PASSWORD", "short")

	if err := run(context.Background(), io.Discard); err == nil || !strings.HasPrefix(err.Error(), "seed administrator:") {
		t.Fatalf("err = %v, want administrator seed error", err)
	}
	var users int
	if err := db.QueryRow(`SELECT count(*) FROM users`).Scan(&users); err != nil || users != 0 {
		t.Fatalf("users = %d, err = %v; want none", users, err)
	}
}
