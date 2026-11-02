package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/medanielsantos/fiap-soat/internal/platform/config"
	"github.com/medanielsantos/fiap-soat/migrations"
	"github.com/pressly/goose/v3"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, args []string, stdout io.Writer) error {
	if len(args) > 1 || (len(args) == 1 && args[0] != "up") {
		return errors.New("usage: go run ./cmd/migrate [up]")
	}

	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		return errors.New("database configuration is required; set DB_NAME and DB_USER in .env")
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("close database: %v", err)
		}
	}()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	return applyMigrations(ctx, db, migrations.Files, stdout)
}

func applyMigrations(ctx context.Context, db *sql.DB, files fs.FS, stdout io.Writer) error {
	provider, err := goose.NewProvider(goose.DialectPostgres, db, files)
	if err != nil {
		return fmt.Errorf("initialize migrations: %w", err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	if len(results) == 0 {
		_, _ = fmt.Fprintln(stdout, "Database is already up to date.")
		return nil
	}
	for _, result := range results {
		_, _ = fmt.Fprintln(stdout, result)
	}
	return nil
}
