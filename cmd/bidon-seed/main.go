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

//go:embed sample_seeds/*.sql
var sampleSeedMigrations embed.FS

func main() {
	reset := flag.Bool("reset", false, "run all DOWN migrations before seeding (clears existing seed data)")
	withSamples := flag.Bool("sample", false, "also run sample test data from sample_seeds/")
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

	ctx := context.Background()

	runSeeds(ctx, db, seedMigrations, "seeds", *reset)

	if *withSamples {
		runSeeds(ctx, db, sampleSeedMigrations, "sample_seeds", *reset)
	}

	log.Println("Done.")
}

func runSeeds(ctx context.Context, db *sql.DB, migrations embed.FS, dir string, reset bool) {
	subFS, err := fs.Sub(migrations, dir)
	if err != nil {
		log.Fatal(err)
	}

	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		db,
		subFS,
		goose.WithDisableVersioning(true),
	)
	if err != nil {
		log.Fatal(err)
	}

	if reset {
		log.Printf("Clearing %s (running DOWN migrations)...", dir)
		results, err := provider.DownTo(ctx, 0)
		if err != nil {
			log.Fatalf("failed to reset %s: %v", dir, err)
		}
		for _, r := range results {
			log.Printf("  down: %s (%s)", r.Source.Path, r.Duration)
		}
	}

	log.Printf("Running %s...", dir)
	results, err := provider.Up(ctx)
	if err != nil {
		log.Fatalf("failed to run %s: %v", dir, err)
	}
	if len(results) == 0 {
		log.Printf("No %s to run (all already applied).", dir)
	}
	for _, r := range results {
		log.Printf("  up: %s (%s)", r.Source.Path, r.Duration)
	}
}
