package main

import (
	"context"
	"fmt"
	"log"

	identitypostgres "github.com/medanielsantos/fiap-soat/internal/identity/adapters/out/postgres"
	identityapp "github.com/medanielsantos/fiap-soat/internal/identity/application"
	"github.com/medanielsantos/fiap-soat/internal/platform/config"
	"github.com/medanielsantos/fiap-soat/internal/platform/database"
	platformid "github.com/medanielsantos/fiap-soat/internal/platform/id"
)

func main() {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Fatal("database configuration is required; set DB_NAME and DB_USER in .env")
	}
	if cfg.AdminName == "" || cfg.AdminEmail == "" || cfg.AdminPassword == "" {
		log.Fatal("ADMIN_NAME, ADMIN_EMAIL and ADMIN_PASSWORD are required")
	}
	db, err := database.OpenPostgres(cfg)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("get database connection: %v", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			log.Printf("close database: %v", err)
		}
	}()

	store := identitypostgres.NewRepository(db)
	if err := store.SeedAuthorization(context.Background()); err != nil {
		log.Fatalf("seed roles and permissions: %v", err)
	}
	service := identityapp.NewService(store, platformid.UUIDGenerator{}, nil, nil, "")
	created, err := service.SeedAdmin(context.Background(), cfg.AdminName, cfg.AdminEmail, cfg.AdminPassword)
	if err != nil {
		log.Fatalf("seed administrator: %v", err)
	}
	if created {
		fmt.Println("Initial administrator created.")
		return
	}
	fmt.Println("The user table is already initialized; seed skipped without changes.")
}
