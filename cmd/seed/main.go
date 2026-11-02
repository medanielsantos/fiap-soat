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

type seedUser struct {
	name  string
	email string
	role  identityapp.UserRole
}

var seedUsers = []seedUser{
	{"Administrador", "admin@oficina.local", identityapp.UserRoleAdmin},
	{"Gerente", "gerente@oficina.local", identityapp.UserRoleManager},
	{"Atendente", "atendente@oficina.local", identityapp.UserRoleAttendant},
}

func main() {
	if err := run(context.Background(), os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, stdout io.Writer) error {
	cfg := config.Load()
	if cfg.AppEnv != config.EnvLocal {
		return errors.New("seed only runs with APP_ENV=local")
	}
	if cfg.DatabaseURL == "" {
		return errors.New("database configuration is required; set DB_NAME and DB_USER in .env")
	}
	if cfg.SeedPassword == "" {
		return errors.New("DEFAULT_PASSWORD_SEED is required")
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
	for _, user := range seedUsers {
		_, err := service.Register(ctx, user.name, user.email, cfg.SeedPassword, []identityapp.UserRole{user.role})
		switch {
		case err == nil:
			_, _ = fmt.Fprintf(stdout, "Created %s (%s).\n", user.email, user.role)
		case identitypostgres.IsUniqueViolation(err):
			_, _ = fmt.Fprintf(stdout, "Skipped %s: already exists.\n", user.email)
		default:
			return fmt.Errorf("seed user %s: %w", user.email, err)
		}
	}
	return nil
}
