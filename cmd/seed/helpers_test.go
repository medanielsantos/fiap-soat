package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/medanielsantos/fiap-soat/migrations"
	"github.com/pressly/goose/v3"
)

func newSchemaDSN(t *testing.T, migrate bool) (string, *sql.DB) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run seed command tests")
	}
	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("cmd_seed_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(`CREATE SCHEMA "` + schema + `"`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(`DROP SCHEMA "` + schema + `" CASCADE`); err != nil {
			t.Errorf("drop schema: %v", err)
		}
		_ = admin.Close()
	})
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("options", "-csearch_path="+schema)
	parsed.RawQuery = query.Encode()
	schemaDSN := parsed.String()

	db, err := sql.Open("pgx", schemaDSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if migrate {
		provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.Files)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := provider.Up(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	return schemaDSN, db
}

func setSeedEnvironment(t *testing.T, dsn string) {
	t.Helper()
	for key, value := range map[string]string{
		"DATABASE_URL": dsn, "DB_NAME": "", "DB_USER": "",
		"ADMIN_NAME": "Administrator", "ADMIN_EMAIL": "admin@example.com", "ADMIN_PASSWORD": "a-strong-password",
	} {
		t.Setenv(key, value)
	}
}
