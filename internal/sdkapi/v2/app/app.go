// Package app assembles the bidon-sdkapi Echo application: stores, caches,
// the auction service, and the v2 router. It owns everything that can be
// constructed from already-open infrastructure (DB, Redis, a logger); it does
// not open that infrastructure itself, so it can be built once in main and
// once per test.
package app

import (
	"fmt"
	"net/http"
	"time"

	"github.com/bool64/cache"
	"github.com/labstack/echo/v4"
	"github.com/oschwald/maxminddb-golang"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.uber.org/zap"

	"github.com/bidon-io/bidon-backend/config"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	adapterstore "github.com/bidon-io/bidon-backend/internal/adapter/store"
	"github.com/bidon-io/bidon-backend/internal/auction"
	auctionstore "github.com/bidon-io/bidon-backend/internal/auction/store"
	"github.com/bidon-io/bidon-backend/internal/bidding"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters_builder"
	dbpkg "github.com/bidon-io/bidon-backend/internal/db"
	"github.com/bidon-io/bidon-backend/internal/notification"
	notificationstore "github.com/bidon-io/bidon-backend/internal/notification/store"
	"github.com/bidon-io/bidon-backend/internal/sdkapi"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/geocoder"
	sdkapistore "github.com/bidon-io/bidon-backend/internal/sdkapi/store"
	v2 "github.com/bidon-io/bidon-backend/internal/sdkapi/v2"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/v2/openapi"
	"github.com/bidon-io/bidon-backend/internal/segment"
	segmentstore "github.com/bidon-io/bidon-backend/internal/segment/store"
	"github.com/bidon-io/bidon-backend/pkg/clock"
)

// Deps are the already-constructed dependencies New wires into the app. DB,
// Redis, EventLogger, and Logger are required; the rest have defaults that
// match production behaviour when left zero.
type Deps struct {
	DB          *dbpkg.DB
	Redis       redis.UniversalClient
	EventLogger *event.Logger
	Logger      *zap.Logger

	// MaxMindDB is optional. A nil reader disables geo lookups; BaseHandler
	// logs and continues with zero GeoData.
	MaxMindDB *maxminddb.Reader
	// HTTPClient is used for outbound bidding requests and win/loss/billing
	// notifications. Defaults to a plain 4s-timeout client.
	HTTPClient *http.Client
	// Meter registers cache-hit/miss observers. Defaults to a noop meter.
	Meter metric.Meter

	// Service names the app in tracing/logging middleware.
	Service string
	// LogRequestAndResponse enables the request/response body dump middleware.
	LogRequestAndResponse bool
}

// App is the assembled bidon-sdkapi application: the configured Echo
// instance, plus the pieces main also needs to build the gRPC server.
type App struct {
	Echo *echo.Echo

	AuctionService *auction.Service
	AppFetcher     *sdkapistore.AppFetcher
	GeoCoder       *geocoder.Geocoder
}

const cacheTTL = 10 * time.Minute

// New builds the Echo application: caches, stores, the auction service, and
// the v2 router, with common middleware and a health check already
// registered. It does not start listening.
func New(deps Deps) (*App, error) {
	if deps.DB == nil {
		return nil, fmt.Errorf("app.New: DB is required")
	}
	if deps.Redis == nil {
		return nil, fmt.Errorf("app.New: Redis is required")
	}
	if deps.EventLogger == nil {
		return nil, fmt.Errorf("app.New: EventLogger is required")
	}
	if deps.Logger == nil {
		return nil, fmt.Errorf("app.New: Logger is required")
	}
	if deps.HTTPClient == nil {
		deps.HTTPClient = &http.Client{Timeout: 4 * time.Second}
	}
	if deps.Meter == nil {
		deps.Meter = noop.Meter{}
	}
	if deps.Service == "" {
		deps.Service = "bidon-sdkapi"
	}

	db := deps.DB
	rdb := deps.Redis
	meter := deps.Meter
	logger := deps.Logger
	eventLogger := deps.EventLogger

	geoCoder := &geocoder.Geocoder{
		DB:        db,
		MaxMindDB: deps.MaxMindDB,
		Cache:     config.NewMemoryCacheOf[*dbpkg.Country](cache.UnlimitedTTL), // We don't update countries
	}
	auctionCache := config.NewRedisCacheOf[*auction.Config](rdb, cacheTTL, "auction_configs")
	if err := auctionCache.Monitor(meter); err != nil {
		return nil, fmt.Errorf("unable to register observer for auctionCache: %w", err)
	}
	configFetcher := &auctionstore.ConfigFetcher{
		DB:    db,
		Cache: auctionCache,
	}
	appCache := config.NewRedisCacheOf[sdkapi.App](rdb, cacheTTL, "apps")
	if err := appCache.Monitor(meter); err != nil {
		return nil, fmt.Errorf("unable to register observer for appCache: %w", err)
	}
	appFetcher := &sdkapistore.AppFetcher{
		DB:    db,
		Cache: appCache,
	}
	segmentCache := config.NewRedisCacheOf[[]segment.Segment](rdb, cacheTTL, "segments")
	if err := segmentCache.Monitor(meter); err != nil {
		return nil, fmt.Errorf("unable to register observer for segmentCache: %w", err)
	}
	segmentMatcher := &segment.Matcher{
		Fetcher: &segmentstore.SegmentFetcher{
			DB:    db,
			Cache: segmentCache,
		},
	}
	notificationHandler := notification.Handler{
		AuctionResultRepo: notificationstore.AuctionResultRepo{Redis: rdb},
		Sender: notification.EventSender{
			HttpClient:  deps.HTTPClient,
			EventLogger: eventLogger,
		},
	}
	adUnitsCache := config.NewRedisCacheOf[[]auction.AdUnit](rdb, cacheTTL, "ad_units")
	if err := adUnitsCache.Monitor(meter); err != nil {
		return nil, fmt.Errorf("unable to register observer for adUnitsCache: %w", err)
	}
	adUnitsMatcher := &auctionstore.AdUnitsMatcher{
		DB:    db,
		Cache: adUnitsCache,
	}
	biddingBuilder := &bidding.Builder{
		AdaptersBuilder:     adapters_builder.BuildBiddingAdapters(deps.HTTPClient),
		NotificationHandler: notificationHandler,
		BidCacher:           &bidding.BidCache{Redis: rdb, Clock: clock.New()},
		Logger:              logger.Named("bidding"),
	}
	biddingAdaptersCfgCache := config.NewRedisCacheOf[adapter.RawConfigsMap](rdb, cacheTTL, "bidding_adapters_cfg")
	if err := biddingAdaptersCfgCache.Monitor(meter); err != nil {
		return nil, fmt.Errorf("unable to register observer for biddingAdaptersCfgCache: %w", err)
	}
	demandCfg := config.NewDemandConfig()
	biddingAdaptersCfgBuilder := adapters_builder.NewAdaptersConfigBuilder(
		&adapterstore.ConfigurationFetcher{
			DB:    db,
			Cache: biddingAdaptersCfgCache,
		},
		demandCfg,
	)
	lineItemsCache := config.NewRedisCacheOf[[]dbpkg.LineItem](rdb, cacheTTL, "line_items")
	if err := lineItemsCache.Monitor(meter); err != nil {
		return nil, fmt.Errorf("unable to register observer for lineItemsCache: %w", err)
	}
	profilesCache := config.NewRedisCacheOf[[]dbpkg.AppDemandProfile](rdb, cacheTTL, "app_demand_profiles")
	if err := profilesCache.Monitor(meter); err != nil {
		return nil, fmt.Errorf("unable to register observer for profilesCache: %w", err)
	}
	amazonSlotsCache := config.NewRedisCacheOf[[]sdkapi.AmazonSlot](rdb, cacheTTL, "amazon_slots")
	if err := amazonSlotsCache.Monitor(meter); err != nil {
		return nil, fmt.Errorf("unable to register observer for amazonSlotsCache: %w", err)
	}

	adapterInitConfigsFetcher := &sdkapistore.AdapterInitConfigsFetcher{DB: db, ProfilesCache: profilesCache, AmazonSlotsCache: amazonSlotsCache, LineItemsCache: lineItemsCache}
	configsCache := config.NewRedisCacheOf[adapter.RawConfigsMap](rdb, cacheTTL, "configs")
	if err := configsCache.Monitor(meter); err != nil {
		return nil, fmt.Errorf("unable to register observer for configsCache: %w", err)
	}
	configurationFetcher := &adapterstore.ConfigurationFetcher{
		DB:    db,
		Cache: configsCache,
	}
	adUnitLookupCache := config.NewRedisCacheOf[*dbpkg.LineItem](rdb, cacheTTL, "ad_unit_lookup")
	if err := adUnitLookupCache.Monitor(meter); err != nil {
		return nil, fmt.Errorf("unable to register observer for adUnitLookupCache: %w", err)
	}
	adUnitLookup := &sdkapistore.AdUnitLookup{
		DB:    db,
		Cache: adUnitLookupCache,
	}
	auctionService := &auction.Service{
		ConfigFetcher:      configFetcher,
		SegmentMatcher:     segmentMatcher,
		AdapterKeysFetcher: adapterInitConfigsFetcher,
		AuctionBuilder: &auction.Builder{
			AdUnitsMatcher:               adUnitsMatcher,
			BiddingBuilder:               biddingBuilder,
			BiddingAdaptersConfigBuilder: biddingAdaptersCfgBuilder,
		},
		EventLogger: eventLogger,
	}

	e := config.Echo()

	v2Group := e.Group("")
	config.UseCommonMiddleware(v2Group, config.Middleware{
		Service:               deps.Service,
		Logger:                logger,
		LogRequestAndResponse: deps.LogRequestAndResponse,
	})
	v2Group.Use(sdkapi.CheckBidonHeader)
	routerV2 := v2.Router{
		ConfigFetcher:             configFetcher,
		AppFetcher:                appFetcher,
		SegmentMatcher:            segmentMatcher,
		BiddingBuilder:            biddingBuilder,
		AdUnitsMatcher:            adUnitsMatcher,
		NotificationHandler:       notificationHandler,
		GeoCoder:                  geoCoder,
		EventLogger:               eventLogger,
		AdapterInitConfigsFetcher: adapterInitConfigsFetcher,
		ConfigurationFetcher:      configurationFetcher,
		AuctionService:            auctionService,
		AdUnitLookup:              adUnitLookup,
	}
	routerV2.RegisterRoutes(v2Group)

	docsWebServer := http.FileServer(http.FS(openapi.FS))
	e.GET("/docs/*", echo.WrapHandler(http.StripPrefix("/docs/", docsWebServer)))

	config.UseHealthCheckHandler(e, config.HealthCheckParams{
		"db":    db,
		"redis": config.NewRedisPinger(rdb),
		"kafka": eventLogger.Engine,
	})

	return &App{
		Echo:           e,
		AuctionService: auctionService,
		AppFetcher:     appFetcher,
		GeoCoder:       geoCoder,
	}, nil
}
