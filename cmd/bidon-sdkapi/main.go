package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	_ "github.com/joho/godotenv/autoload"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/oschwald/maxminddb-golang"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"google.golang.org/grpc/reflection"

	"github.com/bidon-io/bidon-backend/config"
	dbpkg "github.com/bidon-io/bidon-backend/internal/db"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event/engine"
	grpcserver "github.com/bidon-io/bidon-backend/internal/sdkapi/grpc"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/v2/app"
	pb "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/proto/v1"
)

var cpus = runtime.GOMAXPROCS(0)

func main() {
	config.ConfigureOTel()
	exporter, err := prometheus.New()
	if err != nil {
		log.Fatalf("prometheus.New(): %v", err)
	}
	provider := metric.NewMeterProvider(metric.WithReader(exporter))
	meter := provider.Meter("bidon-sdkapi")

	logger, err := config.NewLogger()
	if err != nil {
		log.Fatalf("config.NewLogger(): %v", err)
	}
	defer logger.Sync() //nolint:errcheck

	sentryConf := config.Sentry()
	err = sentry.Init(sentryConf.ClientOptions)
	if err != nil {
		log.Fatalf("sentry.Init(%+v): %v", sentryConf.ClientOptions, err)
	}
	defer sentry.Flush(sentryConf.FlushTimeout)

	dbURL := os.Getenv("DATABASE_REPLICA_URL")
	dbConfig := dbpkg.Config{
		MaxOpenConns:    10 * cpus,
		MaxIdleConns:    5 * cpus,
		ConnMaxLifetime: 15 * time.Minute,
		ReadOnly:        true,
	}
	db, err := dbpkg.Open(dbURL, dbpkg.WithConfig(dbConfig))
	if err != nil {
		log.Fatalf("db.Open(%v): %v", dbURL, err)
	}

	rdb, err := config.NewRedisClient(context.Background(), 10*cpus)
	if err != nil {
		log.Fatalf("config.NewRedisClient(): %v", err)
	}
	{
		pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := rdb.Ping(pingCtx).Err()
		cancel()
		if err != nil {
			log.Fatalf("redis.Ping(): %v", err)
		}
	}

	var maxMindDB *maxminddb.Reader

	if os.Getenv("USE_GEOCODING") == "true" {
		maxMindDB, err = maxminddb.Open(os.Getenv("MAXMIND_GEOIP_FILE_PATH"))
		if err != nil {
			log.Fatalf("maxminddb.Open(%v): %v", os.Getenv("MAXMIND_GEOIP_FILE_PATH"), err)
		}
	}

	var loggerEngine event.LoggerEngine
	if os.Getenv("USE_KAFKA") == "true" {
		conf, err := config.Kafka()
		if err != nil {
			log.Fatalf("config.Kafka(): %v", err)
		}

		client, err := kgo.NewClient(conf.ClientOpts...)
		if err != nil {
			log.Fatalf("kgo.NewClient(): %v", err)
		}
		defer func() {
			ctx, ctxCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer ctxCancel()

			err := client.Flush(ctx)
			if err != nil {
				log.Printf("kgo.Client.Flush(): %v", err)
			}
		}()

		loggerEngine = &engine.Kafka{Client: client, Topics: conf.Topics}
	} else {
		loggerEngine = &engine.Log{}
	}
	eventLogger := &event.Logger{Engine: loggerEngine}

	biddingHTTPClient := &http.Client{
		Timeout: 4 * time.Second,
		Transport: otelhttp.NewTransport(&http.Transport{
			MaxConnsPerHost:     30 * cpus,
			MaxIdleConns:        30 * cpus,
			MaxIdleConnsPerHost: 30 * cpus,
		}),
	}

	sdkapiApp, err := app.New(app.Deps{
		DB:                    db,
		Redis:                 rdb,
		EventLogger:           eventLogger,
		Logger:                logger,
		MaxMindDB:             maxMindDB,
		HTTPClient:            biddingHTTPClient,
		Meter:                 meter,
		Service:               "bidon-sdkapi",
		LogRequestAndResponse: true,
	})
	if err != nil {
		log.Fatalf("app.New(): %v", err)
	}
	e := sdkapiApp.Echo
	auctionService := sdkapiApp.AuctionService
	appFetcher := sdkapiApp.AppFetcher
	geoCoder := sdkapiApp.GeoCoder

	e.Use(echoprometheus.NewMiddleware("sdkapi"))  // adds middleware to gather metrics
	e.GET("/metrics", echoprometheus.NewHandler()) // adds route to serve gathered metrics

	port := os.Getenv("PORT")
	if port == "" {
		port = "1323"
	}
	addr := fmt.Sprintf(":%s", port)

	go func() {
		err := e.Start(addr)
		if !errors.Is(err, http.ErrServerClosed) {
			e.Logger.Fatalf("failed to start http server: %v", err)
		}
		e.Logger.Warn(err)
	}()

	grpcServer := config.NewGRPCServer(logger)
	go func() {
		grpcPort := os.Getenv("GRPC_PORT")
		if grpcPort == "" {
			grpcPort = "50051"
		}
		grpcAddr := fmt.Sprintf(":%s", grpcPort)

		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			log.Fatalf("Failed to listen on %s: %v", grpcAddr, err)
		}

		server := grpcserver.NewServer(auctionService, appFetcher, geoCoder)
		pb.RegisterBiddingServiceServer(grpcServer, server)
		if os.Getenv("ENVIRONMENT") == "development" {
			reflection.Register(grpcServer)
		}

		log.Printf("gRPC server is listening on %s", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Errorf("failed to gracefully shutdown http server: %v", err)
	}

	grpcServer.GracefulStop()
}
