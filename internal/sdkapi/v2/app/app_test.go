package app_test

import (
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	dbpkg "github.com/bidon-io/bidon-backend/internal/db"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event/engine"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/v2/app"
)

// unconnectedRedis returns a client that is never dialed: app.New only wraps
// it in caches at construction time, it never issues a command.
func unconnectedRedis() redis.UniversalClient {
	return redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
}

// TestNew_RegistersExpectedRoutes builds the app with the minimal deps New
// itself never dereferences (no DB query, no Redis command happens during
// construction), and asserts the route table matches what
// cmd/bidon-sdkapi/main.go used to register inline. It exists to catch a
// botched extraction without needing Postgres or Redis.
func TestNew_RegistersExpectedRoutes(t *testing.T) {
	a, err := app.New(app.Deps{
		DB:          &dbpkg.DB{},
		Redis:       unconnectedRedis(),
		EventLogger: &event.Logger{Engine: &engine.Log{}},
		Logger:      zap.NewNop(),
	})
	if err != nil {
		t.Fatalf("app.New() error = %v", err)
	}

	want := map[string]bool{
		"GET /openapi.json":         true,
		"POST /v2/auction/:ad_type": true,
		"POST /v2/click/:ad_type":   true,
		"POST /v2/config":           true,
		"POST /v2/loss/:ad_type":    true,
		"POST /v2/reward/:ad_type":  true,
		"POST /v2/show/:ad_type":    true,
		"POST /v2/stats/:ad_type":   true,
		"POST /v2/win/:ad_type":     true,
		"GET /docs/*":               true,
		"GET /health_checks":        true,
	}

	got := map[string]bool{}
	for _, r := range a.Echo.Routes() {
		got[r.Method+" "+r.Path] = true
	}

	for route := range want {
		if !got[route] {
			t.Errorf("missing expected route %q", route)
		}
	}

	if a.AuctionService == nil {
		t.Error("AuctionService is nil")
	}
	if a.AppFetcher == nil {
		t.Error("AppFetcher is nil")
	}
	if a.GeoCoder == nil {
		t.Error("GeoCoder is nil")
	}
	if a.Echo == nil {
		t.Fatal("Echo is nil")
	}
	if _, ok := interface{}(a.Echo).(*echo.Echo); !ok {
		t.Error("Echo is not *echo.Echo")
	}
}

// TestNew_RequiresCoreDeps asserts New fails fast instead of panicking later
// when a required dependency is missing.
func TestNew_RequiresCoreDeps(t *testing.T) {
	base := app.Deps{
		DB:          &dbpkg.DB{},
		Redis:       unconnectedRedis(),
		EventLogger: &event.Logger{Engine: &engine.Log{}},
		Logger:      zap.NewNop(),
	}

	tests := []struct {
		name   string
		modify func(d app.Deps) app.Deps
	}{
		{"missing DB", func(d app.Deps) app.Deps { d.DB = nil; return d }},
		{"missing Redis", func(d app.Deps) app.Deps { d.Redis = nil; return d }},
		{"missing EventLogger", func(d app.Deps) app.Deps { d.EventLogger = nil; return d }},
		{"missing Logger", func(d app.Deps) app.Deps { d.Logger = nil; return d }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := app.New(tt.modify(base)); err == nil {
				t.Fatal("expected an error, got nil")
			}
		})
	}
}
