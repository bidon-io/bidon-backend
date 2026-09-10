//go:build e2e

// Package e2e drives the assembled bidon-sdkapi Echo application over real
// HTTP, against a real Postgres, a real Redis, and a real OpenRTB DSP
// simulator (bidon-dspsim) — all three provided by docker-compose.test.yml,
// never by this package. Start them with `just test-e2e-up`, then run this
// suite with `just test-e2e` or, containerised, `just test-e2e-ci`.
//
// The suite is excluded from `go test ./...` by the e2e build tag, and it
// intentionally commits its fixtures (dspsim is a separate process reading
// the same database, so a rolled-back transaction would be invisible to it).
// That is safe only because docker-compose.test.yml gives it a database of
// its own that nothing else touches.
package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/db"
	"github.com/bidon-io/bidon-backend/internal/db/dbtest"
	"github.com/bidon-io/bidon-backend/internal/dspsim"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event/engine"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/v2/app"
)

// Tables the fixtures write to. Truncated at TestMain start so reruns against
// an already-running stack (`just test-e2e` twice in a row) start clean.
const fixtureTables = "auction_configurations, line_items, app_demand_profiles, demand_source_accounts, demand_sources, apps, users"

const sdkVersion = "0.6.0"

var (
	testDB     *db.DB
	testRedis  redis.UniversalClient
	dspsimURL  string
	httpClient = &http.Client{Timeout: 10 * time.Second}

	// adikteevSource is created once in TestMain and shared by every fixture:
	// demand_sources.api_key is unique, and it must be literally "adikteev"
	// for the adapter registry to match, so it can't be recreated per test.
	adikteevSource db.DemandSource
)

func databaseURL() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return "postgres://bidon:pass@localhost:5436/bidon_e2e"
}

func redisURL() string {
	if url := os.Getenv("REDIS_URL"); url != "" {
		return url
	}
	return "redis://localhost:6380/0"
}

func dspsimBaseURL() string {
	if url := os.Getenv("DSPSIM_URL"); url != "" {
		return strings.TrimRight(url, "/")
	}
	return "http://localhost:1326"
}

func TestMain(m *testing.M) {
	testDB = dbtest.PrepareURL(databaseURL())

	if err := testDB.Exec(fmt.Sprintf("TRUNCATE %s RESTART IDENTITY CASCADE", fixtureTables)).Error; err != nil {
		fatalf("truncate fixture tables: %v", err)
	}

	adikteevSource = db.DemandSource{APIKey: string(adapter.AdikteevKey), HumanName: "Adikteev"}
	if err := testDB.Create(&adikteevSource).Error; err != nil {
		fatalf("create adikteev demand source: %v", err)
	}

	opt, err := redis.ParseURL(redisURL())
	if err != nil {
		fatalf("parse REDIS_URL %q: %v", redisURL(), err)
	}
	testRedis = redis.NewClient(opt)
	if err := testRedis.Ping(context.Background()).Err(); err != nil {
		fatalf("ping redis at %s: %v. Start the stack: just test-e2e-up", redisURL(), err)
	}
	if err := testRedis.FlushDB(context.Background()).Err(); err != nil {
		fatalf("flush redis: %v", err)
	}

	dspsimURL = dspsimBaseURL()
	if _, err := httpClient.Get(dspsimURL + "/health_checks"); err != nil {
		fatalf("reach dspsim at %s: %v. Start the stack: just test-e2e-up", dspsimURL, err)
	}

	os.Exit(m.Run())
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "e2e: "+format+"\n", args...)
	os.Exit(1)
}

// newServer builds the real sdkapi application and serves it over httptest,
// so routing, middleware, and X-Bidon-Version validation are all exercised
// exactly as in production.
func newServer(t *testing.T) *httptest.Server {
	t.Helper()

	a, err := app.New(app.Deps{
		DB:          testDB,
		Redis:       testRedis,
		EventLogger: &event.Logger{Engine: &engine.Log{}},
		Logger:      zap.NewNop(),
		HTTPClient:  httpClient,
	})
	require.NoError(t, err)

	srv := httptest.NewServer(a.Echo)
	t.Cleanup(srv.Close)
	return srv
}

// postJSON sends body as JSON to srv.URL+path with the headers every sdkapi
// endpoint requires, and decodes the response into out (if non-nil).
func postJSON(t *testing.T, srv *httptest.Server, path string, body, out any) *http.Response {
	t.Helper()

	raw, err := json.Marshal(body)
	require.NoError(t, err)

	// AuctionRequest/ShowRequest carry an untagged `AdType ad.Type
	// \`param:"ad_type"\`` field, so json.Marshal emits it as "AdType" with
	// whatever zero value our Go struct left it at. Echo's DefaultBinder
	// binds path params first and the JSON body last (bind.go:131-143), so a
	// stray "AdType" key in the body would overwrite the value already bound
	// from the URL path. Strip it and let the path param win, same as a real
	// SDK request (which never sends this field at all).
	var stripped map[string]any
	require.NoError(t, json.Unmarshal(raw, &stripped))
	delete(stripped, "AdType")
	raw, err = json.Marshal(stripped)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, srv.URL+path, strings.NewReader(string(raw)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Bidon-Version", sdkVersion)

	resp, err := httpClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })

	if out != nil {
		require.NoError(t, json.NewDecoder(resp.Body).Decode(out))
	}
	return resp
}

// dspsimReload forces dspsim to reread the catalog from Postgres, so a
// fixture committed a moment ago is visible without waiting on
// DSPSIM_CATALOG_TTL.
func dspsimReload(t *testing.T) {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, dspsimURL+"/debug/reload", nil)
	require.NoError(t, err)

	resp, err := httpClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// dspsimBid fetches a single bid record by id. ok is false on a 404.
func dspsimBid(t *testing.T, bidID string) (dspsim.BidRecord, bool) {
	t.Helper()

	resp, err := httpClient.Get(dspsimURL + "/debug/bids/" + bidID)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return dspsim.BidRecord{}, false
	}
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var record dspsim.BidRecord
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&record))
	return record, true
}

// dspsimFindBid returns the Adikteev bid dspsim recorded for bundle. The
// auction response never exposes dspsim's bid id (only the OpenRTB payload
// does, buried in ext.payload), so tests correlate by the fixture's unique
// bundle instead.
func dspsimFindBid(t *testing.T, bundle string) dspsim.BidRecord {
	t.Helper()

	resp, err := httpClient.Get(dspsimURL + "/debug/bids?dsp=adikteev")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body struct {
		Bids []dspsim.BidRecord `json:"bids"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	for _, b := range body.Bids {
		if b.Bundle == bundle {
			return b
		}
	}
	t.Fatalf("no dspsim bid recorded for bundle %q", bundle)
	return dspsim.BidRecord{}
}

// waitForNotification polls dspsim for bidID until it has recorded a
// notification of kind (see dspsim.Notification*Win/Loss/Billing consts), or
// fails the test after timeout. Notifications are fired from detached
// goroutines after the SDK API request returns, so callers must poll rather
// than assert synchronously.
func waitForNotification(t *testing.T, bidID, kind string, timeout time.Duration) dspsim.Notification {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for {
		record, ok := dspsimBid(t, bidID)
		if ok {
			for _, n := range record.Notifications {
				if n.Kind == kind {
					return n
				}
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for a %q notification on bid %s", kind, bidID)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
