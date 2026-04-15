package main

import (
	"context"
	"database/sql"
	"embed"
	"flag"
	"io/fs"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/bidon-io/bidon-backend/config"
)

//go:embed seeds/*.sql
var seedMigrations embed.FS

func main() {
	reset := flag.Bool("reset", false, "run all DOWN migrations before seeding (clears existing seed data)")
	flag.Parse()

	config.LoadEnvFile()

	if config.GetEnv() == config.TestEnv {
		log.Println("Skipping seeds in test environment.")
		return
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("missing DATABASE_URL environment variable")
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Fatal("failed to open DB: ", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Fatal("failed to close DB: ", err)
		}
	}()

	seedsFS, err := fs.Sub(seedMigrations, "seeds")
	if err != nil {
		log.Fatal(err)
	}

	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		db,
		seedsFS,
		goose.WithDisableVersioning(true),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	if *reset {
		log.Println("Clearing seed data (running DOWN migrations)...")
		results, err := provider.DownTo(ctx, 0)
		if err != nil {
			log.Fatal("failed to reset seeds: ", err)
		}
		for _, r := range results {
			log.Printf("  down: %s (%s)", r.Source.Path, r.Duration)
		}
	}

	log.Println("Running seeds...")
	results, err := provider.Up(ctx)
	if err != nil {
		log.Fatal("failed to run seeds: ", err)
	}
	if len(results) == 0 {
		log.Println("No seeds to run (all already applied).")
	}
	for _, r := range results {
		log.Printf("  up: %s (%s)", r.Source.Path, r.Duration)
	}
	log.Println("Done.")
}
