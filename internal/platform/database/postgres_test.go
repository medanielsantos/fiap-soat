package database

import (
	"os"
	"strings"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/platform/config"
)

func TestOpenPostgresRequiresDatabaseURL(t *testing.T) {
	if _, err := OpenPostgres(config.Config{}); err == nil || err.Error() != "DATABASE_URL is required" {
		t.Fatalf("err = %v, want DATABASE_URL is required", err)
	}
}

func TestOpenPostgresWrapsConnectionFailure(t *testing.T) {
	_, err := OpenPostgres(config.Config{DatabaseURL: "postgres://nobody:nothing@127.0.0.1:1/none?sslmode=disable&connect_timeout=1"})
	if err == nil || !strings.HasPrefix(err.Error(), "open PostgreSQL connection:") {
		t.Fatalf("err = %v, want wrapped connection error", err)
	}
}

func TestOpenPostgresConnects(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL tests")
	}
	db, err := OpenPostgres(config.Config{DatabaseURL: dsn})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	var one int
	if err := db.Raw("SELECT 1").Scan(&one).Error; err != nil || one != 1 {
		t.Fatalf("SELECT 1 = %d, %v", one, err)
	}
}
