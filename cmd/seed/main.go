package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	identitypostgres "github.com/medanielsantos/fiap-soat/internal/identity/adapters/out/postgres"
	identityapp "github.com/medanielsantos/fiap-soat/internal/identity/application"
	"github.com/medanielsantos/fiap-soat/internal/platform/config"
	"github.com/medanielsantos/fiap-soat/internal/platform/database"
	platformid "github.com/medanielsantos/fiap-soat/internal/platform/id"
)

func main() {
	if err := run(context.Background(), os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, stdout io.Writer) error {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		return errors.New("database configuration is required; set DB_NAME and DB_USER in .env")
	}
	if cfg.AdminName == "" || cfg.AdminEmail == "" || cfg.AdminPassword == "" {
		return errors.New("ADMIN_NAME, ADMIN_EMAIL and ADMIN_PASSWORD are required")
	}
	db, err := database.OpenPostgres(cfg)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get database connection: %w", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			log.Printf("close database: %v", err)
		}
	}()

	store := identitypostgres.NewRepository(db)
	if err := store.SeedAuthorization(ctx); err != nil {
		return fmt.Errorf("seed roles and permissions: %w", err)
	}
	service := identityapp.NewService(store, platformid.UUIDGenerator{}, nil, nil, "")
	created, err := service.SeedAdmin(ctx, cfg.AdminName, cfg.AdminEmail, cfg.AdminPassword)
	if err != nil {
		return fmt.Errorf("seed administrator: %w", err)
	}
	if created {
		_, _ = fmt.Fprintln(stdout, "Initial administrator created.")
		return nil
	}
	_, _ = fmt.Fprintln(stdout, "The user table is already initialized; seed skipped without changes.")
	return nil
}
