package main

import (
	"fmt"
	"log"
	"os"

	"github.com/bidon-io/bidon-backend/internal/auction"
	auctionstore "github.com/bidon-io/bidon-backend/internal/auction/store"
	"github.com/bidon-io/bidon-backend/internal/db"
	"github.com/bidon-io/bidon-backend/internal/echoconf"
	"github.com/bidon-io/bidon-backend/internal/sdkapi"
	sdkapistore "github.com/bidon-io/bidon-backend/internal/sdkapi/store"
	"github.com/bidon-io/bidon-backend/internal/sentryconf"
	"github.com/getsentry/sentry-go"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	err := sentry.Init(sentryconf.ClientOptions)
	if err != nil {
		log.Fatalf("sentry.Init(%+v): %v", sentryconf.ClientOptions, err)
	}
	defer sentry.Flush(sentryconf.FlushTimeout)

	dbURL := os.Getenv("DATABASE_URL")
	db, err := db.Open(dbURL)
	if err != nil {
		log.Fatalf("db.Open(%v): %v", dbURL, err)
	}

	service := sdkapi.Service{
		AuctionBuilder: &auction.Builder{
			ConfigMatcher:    &auctionstore.ConfigMatcher{DB: db},
			LineItemsMatcher: &auctionstore.LineItemsMatcher{DB: db},
		},
		AppFetcher: &sdkapistore.AppFetcher{DB: db},
	}

	e := echoconf.NewEcho()

	e.Use(sdkapi.CheckBidonHeader)

	e.POST("/auction/:ad_type", service.HandleAuction)
	e.POST("/:ad_type/auction", service.HandleAuction)

	port := os.Getenv("PORT")
	if port == "" {
		port = "1323"
	}
	addr := fmt.Sprintf(":%s", port)
	e.Logger.Fatal(e.Start(addr))
}
