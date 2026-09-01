package dspsim

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prebid/openrtb/v19/openrtb2"
	"go.uber.org/zap"
)

// newTestServer wires the simulator over an httptest server whose own address
// is advertised in notification URLs, so the notification round trip is real.
func newTestServer(t *testing.T) (*httptest.Server, *Server) {
	t.Helper()

	cfg := testConfig()
	cfg.MaxBids = 100
	cfg.BidTTL = time.Hour

	sim := &Server{
		Config:  cfg,
		Logger:  zap.NewNop(),
		Catalog: testCatalogStore(),
		Bidder:  NewBidder(cfg, loadDefaultLibrary(t)),
		Store:   NewStore(cfg.BidTTL, cfg.MaxBids),
		Matcher: &Matcher{Catalog: testCatalogStore(), MaxPrice: cfg.MaxPrice},
	}

	e := echo.New()
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		e.DefaultHTTPErrorHandler(err, c)
	}
	sim.RegisterRoutes(e.Group(""))

	httpServer := httptest.NewServer(e)
	t.Cleanup(httpServer.Close)

	sim.Config.PublicURL = httpServer.URL
	sim.Bidder.Config.PublicURL = httpServer.URL

	return httpServer, sim
}

func postFixture(t *testing.T, server *httptest.Server, fixture, query string) *http.Response {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(fixtureDir, fixture))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	endpoint := server.URL + "/openrtb/bid"
	if query != "" {
		endpoint += "?" + query
	}

	resp, err := server.Client().Post(endpoint, echo.MIMEApplicationJSON, strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("POST %s: %v", endpoint, err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func decodeBidResponse(t *testing.T, resp *http.Response) *openrtb2.BidResponse {
	t.Helper()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d (%s), want 200", resp.StatusCode, body)
	}

	var bidResponse openrtb2.BidResponse
	if err := json.NewDecoder(resp.Body).Decode(&bidResponse); err != nil {
		t.Fatalf("decode bid response: %v", err)
	}
	return &bidResponse
}

// The full lifecycle: bid, then fire the advertised nurl and burl and check
// they land on the right record with their macros resolved.
func TestBidAndNotificationLifecycle(t *testing.T) {
	server, sim := newTestServer(t)

	bidResponse := decodeBidResponse(t, postFixture(t, server, "android_adikteev_banner_bidreq.json", ""))
	bid := bidResponse.SeatBid[0].Bid[0]

	if bidResponse.SeatBid[0].Seat != "dspsim" {
		t.Errorf("seat = %q, want dspsim", bidResponse.SeatBid[0].Seat)
	}

	// Substitute the macros the way internal/notification/event_sender.go does.
	fire(t, server, bid.NURL, map[string]string{
		MacroPrice: "2.5", MacroMinToWin: "1.1", MacroAuctionID: bidResponse.ID,
		MacroBidID: bid.ID, MacroImpID: bid.ImpID, MacroSeatID: "dspsim",
		MacroAdID: bid.AdID, MacroCurrency: "USD",
	})
	fire(t, server, bid.BURL, map[string]string{
		MacroPrice: "2.5", MacroBidID: bid.ID, MacroImpID: bid.ImpID, MacroCurrency: "USD",
	})

	record, ok := sim.Store.Get(bid.ID)
	if !ok {
		t.Fatalf("no record stored for bid %s", bid.ID)
	}
	if len(record.Notifications) != 2 {
		t.Fatalf("notifications = %d, want 2", len(record.Notifications))
	}

	kinds := map[string]Notification{}
	for _, n := range record.Notifications {
		kinds[n.Kind] = n
	}

	win, ok := kinds[NotificationWin]
	if !ok {
		t.Fatal("no win notification recorded")
	}
	if win.Params["price"] != "2.5" {
		t.Errorf("win price param = %q, want 2.5", win.Params["price"])
	}
	if win.Params["mintowin"] != "1.1" {
		t.Errorf("win mintowin param = %q, want 1.1", win.Params["mintowin"])
	}
	if len(win.UnresolvedMacros) != 0 {
		t.Errorf("unresolved macros = %v, want none", win.UnresolvedMacros)
	}
	if _, ok := kinds[NotificationBilling]; !ok {
		t.Error("no billing notification recorded")
	}
}

// A sender that forgets to substitute must be caught, not silently accepted.
func TestUnresolvedMacrosAreFlagged(t *testing.T) {
	server, sim := newTestServer(t)

	bidResponse := decodeBidResponse(t, postFixture(t, server, "android_adikteev_banner_bidreq.json", ""))
	bid := bidResponse.SeatBid[0].Bid[0]

	// Fire the nurl verbatim, macros and all.
	resp, err := server.Client().Get(bid.NURL)
	if err != nil {
		t.Fatalf("GET nurl: %v", err)
	}
	defer resp.Body.Close()

	record, ok := sim.Store.Get(bid.ID)
	if !ok {
		t.Fatal("no record stored")
	}
	if len(record.Notifications) != 1 {
		t.Fatalf("notifications = %d, want 1", len(record.Notifications))
	}

	unresolved := record.Notifications[0].UnresolvedMacros
	if len(unresolved) < 8 {
		t.Errorf("unresolved macros = %v, want every nurl param flagged", unresolved)
	}
}

func TestNotificationForUnknownBidIsAnOrphan(t *testing.T) {
	server, sim := newTestServer(t)

	resp, err := server.Client().Get(server.URL + "/notify/loss/does-not-exist?price=1.0")
	if err != nil {
		t.Fatalf("GET loss: %v", err)
	}
	defer resp.Body.Close()

	// A real DSP answers 200 regardless.
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	orphans := sim.Store.Orphans()
	if len(orphans) != 1 || orphans[0].BidID != "does-not-exist" {
		t.Fatalf("orphans = %+v, want one for does-not-exist", orphans)
	}
}

func TestCreativeTrackingIsRecorded(t *testing.T) {
	server, sim := newTestServer(t)

	bidResponse := decodeBidResponse(t, postFixture(t, server, "android_adikteev_rewarded_bidreq.json", "creative=default_vast_video"))
	bid := bidResponse.SeatBid[0].Bid[0]

	for _, path := range []string{
		"/creative/impression/" + bid.ID,
		"/creative/click/" + bid.ID,
		"/creative/track/" + bid.ID + "/firstQuartile",
	} {
		resp, err := server.Client().Get(server.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
	}

	record, ok := sim.Store.Get(bid.ID)
	if !ok {
		t.Fatal("no record stored")
	}
	if len(record.Notifications) != 3 {
		t.Fatalf("notifications = %d, want 3", len(record.Notifications))
	}

	var tracked bool
	for _, n := range record.Notifications {
		if n.Kind == NotificationTrack && n.Event == "firstQuartile" {
			tracked = true
		}
	}
	if !tracked {
		t.Errorf("firstQuartile tracking not recorded: %+v", record.Notifications)
	}
	if record.CreativeType != TypeVASTVideo {
		t.Errorf("CreativeType = %q, want %q", record.CreativeType, TypeVASTVideo)
	}
}

func TestNoBidCarriesReasonHeader(t *testing.T) {
	server, _ := newTestServer(t)

	resp, err := server.Client().Post(server.URL+"/openrtb/bid", echo.MIMEApplicationJSON,
		strings.NewReader(`{"id":"r1","app":{"bundle":"com.unknown.app"},"imp":[{"id":"i1","banner":{"w":320,"h":50},"bidfloor":0.1}]}`))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want 204", resp.StatusCode)
	}
	if got := resp.Header.Get("X-Dspsim-Nobid-Reason"); got != string(ReasonUnknownBundle) {
		t.Errorf("no-bid reason header = %q, want %q", got, ReasonUnknownBundle)
	}
}

func TestMalformedRequestIsRejected(t *testing.T) {
	server, _ := newTestServer(t)

	resp, err := server.Client().Post(server.URL+"/openrtb/bid", echo.MIMEApplicationJSON, strings.NewReader("not json"))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestDSPOverrideSelectsBucket(t *testing.T) {
	server, sim := newTestServer(t)

	bidResponse := decodeBidResponse(t, postFixture(t, server, "android_adikteev_banner_bidreq.json", "dsp=does_not_exist"))
	bid := bidResponse.SeatBid[0].Bid[0]

	record, ok := sim.Store.Get(bid.ID)
	if !ok {
		t.Fatal("no record stored")
	}
	if record.DSP != "does_not_exist" {
		t.Errorf("DSP = %q, want does_not_exist", record.DSP)
	}
	if record.CreativeBucket != DefaultDSPBucket {
		t.Errorf("CreativeBucket = %q, want %q", record.CreativeBucket, DefaultDSPBucket)
	}
	if record.DemandConfigured {
		t.Error("DemandConfigured = true for a DSP absent from the auction")
	}
}

func TestDebugEndpoints(t *testing.T) {
	server, _ := newTestServer(t)

	bidResponse := decodeBidResponse(t, postFixture(t, server, "android_adikteev_banner_bidreq.json", ""))
	bidID := bidResponse.SeatBid[0].Bid[0].ID

	t.Run("bids", func(t *testing.T) {
		var payload struct {
			Count int          `json:"count"`
			Bids  []*BidRecord `json:"bids"`
		}
		getJSON(t, server, "/debug/bids", &payload)

		if payload.Count != 1 || len(payload.Bids) != 1 || payload.Bids[0].BidID != bidID {
			t.Fatalf("unexpected /debug/bids payload: %+v", payload)
		}
	})

	t.Run("single bid", func(t *testing.T) {
		var record BidRecord
		getJSON(t, server, "/debug/bids/"+bidID, &record)

		if record.BidID != bidID {
			t.Errorf("BidID = %q, want %q", record.BidID, bidID)
		}
	})

	t.Run("unknown bid", func(t *testing.T) {
		resp, err := server.Client().Get(server.URL + "/debug/bids/nope")
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d, want 404", resp.StatusCode)
		}
	})

	t.Run("creatives", func(t *testing.T) {
		var payload struct {
			Source    string                       `json:"source"`
			Buckets   []string                     `json:"buckets"`
			Creatives map[string][]CreativeSummary `json:"creatives"`
		}
		getJSON(t, server, "/debug/creatives", &payload)

		if len(payload.Buckets) < 2 || len(payload.Creatives) < 2 {
			t.Fatalf("unexpected /debug/creatives payload: %+v", payload)
		}
	})

	t.Run("catalog", func(t *testing.T) {
		var payload struct {
			Bundles  []string             `json:"bundles"`
			Auctions []*ConfiguredAuction `json:"auctions"`
		}
		getJSON(t, server, "/debug/catalog", &payload)

		if len(payload.Bundles) != 1 || payload.Bundles[0] != "com.demo.tetris" {
			t.Errorf("bundles = %v", payload.Bundles)
		}
		if len(payload.Auctions) != 3 {
			t.Errorf("auctions = %d, want 3", len(payload.Auctions))
		}
	})

	t.Run("clear", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodDelete, server.URL+"/debug/bids", nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		resp, err := server.Client().Do(request)
		if err != nil {
			t.Fatalf("DELETE: %v", err)
		}
		defer resp.Body.Close()

		var payload struct {
			Count int `json:"count"`
		}
		getJSON(t, server, "/debug/bids", &payload)
		if payload.Count != 0 {
			t.Errorf("count = %d after DELETE, want 0", payload.Count)
		}
	})
}

func TestAssetHandlerGeneratesSVG(t *testing.T) {
	server, _ := newTestServer(t)

	resp, err := server.Client().Get(server.URL + "/creative/asset/banner-300x250.svg")
	if err != nil {
		t.Fatalf("GET asset: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get(echo.HeaderContentType); !strings.Contains(got, "image/svg+xml") {
		t.Errorf("content type = %q, want image/svg+xml", got)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `width="300"`) || !strings.Contains(string(body), `height="250"`) {
		t.Errorf("asset did not honour the size in its name: %s", body)
	}
}

// fire substitutes macros the way the bidon event sender does — whole-value
// matches only — and issues the GET.
func fire(t *testing.T, server *httptest.Server, rawURL string, macros map[string]string) {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse %q: %v", rawURL, err)
	}

	values, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		t.Fatalf("parse query %q: %v", parsed.RawQuery, err)
	}
	for key := range values {
		if replacement, ok := macros[values.Get(key)]; ok {
			values.Set(key, replacement)
		}
	}
	parsed.RawQuery = values.Encode()

	resp, err := server.Client().Get(parsed.String())
	if err != nil {
		t.Fatalf("GET %s: %v", parsed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET %s: status = %d, want 200", parsed, resp.StatusCode)
	}
}

func getJSON(t *testing.T, server *httptest.Server, path string, out any) {
	t.Helper()

	resp, err := server.Client().Get(server.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET %s: status = %d (%s)", path, resp.StatusCode, body)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}
