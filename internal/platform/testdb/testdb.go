// Package testdb provides an isolated, migrated PostgreSQL schema for repository tests.
package testdb

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
	postgresdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open returns a gorm connection bound to a fresh schema with all migrations applied.
// The test is skipped when TEST_DATABASE_URL is not set; the schema is dropped on cleanup.
func Open(t testing.TB) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL repository tests")
	}

	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open admin connection: %v", err)
	}
	admin.SetMaxOpenConns(1)
	schema := fmt.Sprintf("repo_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(`CREATE SCHEMA "` + schema + `"`); err != nil {
		_ = admin.Close()
		t.Fatalf("create isolated test schema: %v", err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(`DROP SCHEMA "` + schema + `" CASCADE`); err != nil {
			t.Errorf("drop isolated test schema: %v", err)
		}
		_ = admin.Close()
	})

	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	query := parsed.Query()
	query.Set("options", "-csearch_path="+schema)
	parsed.RawQuery = query.Encode()
	schemaDSN := parsed.String()

	sqlDB, err := sql.Open("pgx", schemaDSN)
	if err != nil {
		t.Fatalf("open schema connection: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.Files)
	if err != nil {
		t.Fatalf("create migration provider: %v", err)
	}
	if _, err := provider.Up(context.Background()); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	db, err := gorm.Open(postgresdriver.Open(schemaDSN), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open gorm connection: %v", err)
	}
	gormSQL, err := db.DB()
	if err != nil {
		t.Fatalf("access gorm connection: %v", err)
	}
	t.Cleanup(func() { _ = gormSQL.Close() })
	return db
}
