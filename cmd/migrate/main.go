package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/medanielsantos/fiap-soat/internal/platform/config"
	"github.com/medanielsantos/fiap-soat/migrations"
	"github.com/pressly/goose/v3"
)

func main() {
	if len(os.Args) > 2 || (len(os.Args) == 2 && os.Args[1] != "up") {
		log.Fatal("usage: go run ./cmd/migrate [up]")
	}

	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Fatal("database configuration is required; set DB_NAME and DB_USER in .env")
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("close database: %v", err)
		}
	}()

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.Files)
	if err != nil {
		log.Fatalf("initialize migrations: %v", err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		log.Fatalf("apply migrations: %v", err)
	}
	if len(results) == 0 {
		fmt.Println("Database is already up to date.")
		return
	}
	for _, result := range results {
		fmt.Println(result)
	}
}
