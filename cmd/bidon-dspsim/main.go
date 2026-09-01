// Command bidon-dspsim runs a standalone OpenRTB DSP simulator.
//
// It reads auction configuration from the Bidon Postgres schema (read-only),
// answers bid requests with a random creative drawn from a JSON library indexed
// by DSP and creative type, and receives the nurl / burl / lurl notifications it
// advertises, keeping every interaction in memory keyed by bid id.
//
// See docs/adr/0001-dsp-simulator.md.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/bidon-io/bidon-backend/config"
	dbpkg "github.com/bidon-io/bidon-backend/internal/db"
	"github.com/bidon-io/bidon-backend/internal/dspsim"
)

var cpus = runtime.GOMAXPROCS(0)

func main() {
	config.LoadEnvFile()

	cfg, err := dspsim.LoadConfig()
	if err != nil {
		log.Fatalf("dspsim.LoadConfig(): %v", err)
	}

	logger, err := config.NewLogger()
	if err != nil {
		log.Fatalf("config.NewLogger(): %v", err)
	}
	defer logger.Sync() //nolint:errcheck

	db, err := dbpkg.Open(cfg.DatabaseURL, dbpkg.WithConfig(dbpkg.Config{
		MaxOpenConns:    2 * cpus,
		MaxIdleConns:    cpus,
		ConnMaxLifetime: 15 * time.Minute,
		ReadOnly:        true,
	}))
	if err != nil {
		log.Fatalf("db.Open(): %v", err)
	}

	library, err := dspsim.LoadLibrary(cfg.CreativesFile)
	if err != nil {
		log.Fatalf("dspsim.LoadLibrary(): %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	catalog := dspsim.NewCatalogStore(db, logger)
	if err := catalog.Refresh(ctx); err != nil {
		log.Fatalf("dspsim: initial catalog load failed: %v", err)
	}
	go catalog.Run(ctx, cfg.CatalogTTL)

	server := &dspsim.Server{
		Config:  cfg,
		Logger:  logger,
		Catalog: catalog,
		Bidder:  dspsim.NewBidder(cfg, library),
		Store:   dspsim.NewStore(cfg.BidTTL, cfg.MaxBids),
		Matcher: &dspsim.Matcher{
			Catalog:      catalog,
			MaxPrice:     cfg.MaxPrice,
			StrictDemand: cfg.StrictDemand,
		},
	}

	e := config.Echo()

	group := e.Group("")
	config.UseCommonMiddleware(group, config.Middleware{
		Service:               "bidon-dspsim",
		Logger:                logger,
		LogRequestAndResponse: true,
	})
	server.RegisterRoutes(group)

	config.UseHealthCheckHandler(e, config.HealthCheckParams{"db": db})

	addr := fmt.Sprintf(":%s", cfg.Port)
	go func() {
		log.Printf("dspsim listening on %s, advertising %s (creatives: %s)", addr, cfg.PublicURL, library.Source)
		err := e.Start(addr)
		if !errors.Is(err, http.ErrServerClosed) {
			e.Logger.Fatalf("failed to start http server: %v", err)
		}
		e.Logger.Warn(err)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		e.Logger.Errorf("failed to gracefully shutdown http server: %v", err)
	}
}
