package main

import (
	"context"
	"database/sql"
	"io"
	"strings"
	"testing"
	"testing/fstest"
)

func TestRunRejectsUnknownArguments(t *testing.T) {
	for _, args := range [][]string{{"down"}, {"up", "extra"}} {
		if err := run(context.Background(), args, io.Discard); err == nil || !strings.HasPrefix(err.Error(), "usage:") {
			t.Fatalf("run(%v) = %v, want usage error", args, err)
		}
	}
}

func TestRunRequiresDatabaseConfiguration(t *testing.T) {
	setDatabaseURL(t, "")
	err := run(context.Background(), nil, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "database configuration is required") {
		t.Fatalf("err = %v, want configuration error", err)
	}
}

func TestRunWrapsDatabaseFailures(t *testing.T) {
	cases := []struct{ name, dsn, want string }{
		{"invalid dsn", "postgres://user@host:notaport/db", "connect to database:"},
		{"unreachable", "postgres://nobody@127.0.0.1:1/none?sslmode=disable&connect_timeout=1", "connect to database:"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setDatabaseURL(t, tc.dsn)
			if err := run(context.Background(), nil, io.Discard); err == nil || !strings.HasPrefix(err.Error(), tc.want) {
				t.Fatalf("err = %v, want prefix %q", err, tc.want)
			}
		})
	}
}

func TestRunWrapsMigrationFailure(t *testing.T) {
	dsn, admin := newSchemaDSN(t)
	setDatabaseURL(t, dsn)
	schema := strings.TrimPrefix(dsnOption(t, dsn), "-csearch_path=")
	if _, err := admin.Exec(`CREATE TABLE "` + schema + `".customers (conflict INT)`); err != nil {
		t.Fatal(err)
	}

	if err := run(context.Background(), nil, io.Discard); err == nil || !strings.HasPrefix(err.Error(), "apply migrations:") {
		t.Fatalf("err = %v, want migration failure", err)
	}
}

func TestApplyMigrationsWrapsProviderInitializationFailure(t *testing.T) {
	db, err := sql.Open("pgx", "postgres://unused@127.0.0.1:1/none")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	err = applyMigrations(context.Background(), db, fstest.MapFS{}, io.Discard)
	if err == nil || !strings.HasPrefix(err.Error(), "initialize migrations:") {
		t.Fatalf("err = %v, want provider initialization error", err)
	}
}
