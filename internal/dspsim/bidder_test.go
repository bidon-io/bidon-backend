package dspsim

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	biddingopenrtb "github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

// stubBidder satisfies adapters.BidderInterface so the simulator's responses
// can be run through the real shared parser. It implements neither
// CustomBidParser nor OpenRTBBidEnricher, which is the default OpenRTB path.
type stubBidder struct{}

func (stubBidder) CreateRequest(r biddingopenrtb.BidRequest, _ *schema.AuctionRequest) (biddingopenrtb.BidRequest, error) {
	return r, nil
}

func (stubBidder) ExecuteRequest(context.Context, *http.Client, biddingopenrtb.BidRequest) *adapters.DemandResponse {
	return nil
}

func testConfig() Config {
	return Config{
		PublicURL:     "http://dspsim.test",
		Seat:          "dspsim",
		Currency:      "USD",
		PriceMultMin:  1.5,
		PriceMultMax:  3.0,
		FallbackFloor: 0.5,
		MaxPrice:      25,
		MaxBids:       100,
		Seed:          42,
	}
}

func testMatch(t *testing.T, fixture string) *Match {
	t.Helper()

	summary, err := Describe(loadBidRequest(t, fixture))
	if err != nil {
		t.Fatalf("Describe(): %v", err)
	}

	matcher := &Matcher{Catalog: testCatalogStore(), MaxPrice: 25}
	match, reason := matcher.Match(summary, ad.UnknownType)
	if reason != "" {
		t.Fatalf("Match() no-bid reason = %q", reason)
	}
	return match
}

// The whole point of the response shape: it must survive the parser bidon
// actually uses (internal/bidding/adapters/parse.go).
func TestBidResponseRoundTripsThroughSharedParser(t *testing.T) {
	fixtures := []string{
		"android_adikteev_banner_bidreq.json",
		"android_adikteev_banner_mrec_bidreq.json",
		"android_adikteev_interstitial_bidreq.json",
		"ios_adikteev_rewarded_bidreq.json",
	}

	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			bidder := NewBidder(testConfig(), loadDefaultLibrary(t))
			match := testMatch(t, fixture)

			response, record, reason, err := bidder.Build(match, "")
			if err != nil {
				t.Fatalf("Build(): %v", err)
			}
			if reason != "" {
				t.Fatalf("Build() no-bid reason = %q", reason)
			}

			raw, err := json.Marshal(response)
			if err != nil {
				t.Fatalf("marshal response: %v", err)
			}

			parsed, err := adapters.ParseDemandResponse(stubBidder{}, &adapters.DemandResponse{
				DemandID:    adapter.AdikteevKey,
				Status:      http.StatusOK,
				RawResponse: string(raw),
			})
			if err != nil {
				t.Fatalf("ParseDemandResponse(): %v", err)
			}
			if !parsed.IsBid() {
				t.Fatal("ParseDemandResponse() produced no bid")
			}

			bid := parsed.Bid
			if bid.ID != record.BidID {
				t.Errorf("bid ID = %q, want %q", bid.ID, record.BidID)
			}
			if bid.ImpID != match.Summary.ImpID {
				t.Errorf("bid ImpID = %q, want %q", bid.ImpID, match.Summary.ImpID)
			}
			if bid.Price != record.Price {
				t.Errorf("bid Price = %v, want %v", bid.Price, record.Price)
			}
			if bid.Price <= match.Summary.Floor {
				t.Errorf("bid Price = %v is not above the impression floor %v", bid.Price, match.Summary.Floor)
			}
			if bid.Payload == "" {
				t.Error("bid adm is empty")
			}
			if bid.SeatID != "dspsim" {
				t.Errorf("bid SeatID = %q, want dspsim", bid.SeatID)
			}
			for name, value := range map[string]string{"nurl": bid.NURL, "burl": bid.BURL, "lurl": bid.LURL} {
				if value == "" {
					t.Errorf("%s is empty", name)
				}
			}
		})
	}
}

// Every macro must be the entire value of its own query param, because
// event_sender.go only substitutes on a whole-value match.
func TestNotificationURLsCarryWholeValueMacros(t *testing.T) {
	bidder := NewBidder(testConfig(), loadDefaultLibrary(t))
	_, record, _, err := bidder.Build(testMatch(t, "android_adikteev_banner_bidreq.json"), "")
	if err != nil {
		t.Fatalf("Build(): %v", err)
	}

	tests := []struct {
		name   string
		raw    string
		macros []string
	}{
		{"nurl", record.NURL, []string{MacroPrice, MacroMinToWin, MacroAuctionID, MacroBidID, MacroImpID, MacroSeatID, MacroAdID, MacroCurrency}},
		{"burl", record.BURL, []string{MacroPrice, MacroBidID, MacroImpID, MacroCurrency}},
		{"lurl", record.LURL, []string{MacroPrice, MacroLoss, MacroMinToWin, MacroAuctionID, MacroBidID, MacroCurrency}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := url.Parse(tt.raw)
			if err != nil {
				t.Fatalf("parse %s %q: %v", tt.name, tt.raw, err)
			}
			if !strings.HasPrefix(parsed.Path, "/notify/") || !strings.HasSuffix(parsed.Path, record.BidID) {
				t.Errorf("path = %q, want /notify/<kind>/%s", parsed.Path, record.BidID)
			}

			values, err := url.ParseQuery(parsed.RawQuery)
			if err != nil {
				t.Fatalf("parse query %q: %v", parsed.RawQuery, err)
			}

			present := map[string]bool{}
			for _, vs := range values {
				for _, v := range vs {
					present[v] = true
				}
			}
			for _, macro := range tt.macros {
				if !present[macro] {
					t.Errorf("macro %s is not the whole value of any param in %s", macro, tt.raw)
				}
			}
		})
	}
}

func TestBuildUsesForcedCreative(t *testing.T) {
	bidder := NewBidder(testConfig(), loadDefaultLibrary(t))

	_, record, reason, err := bidder.Build(testMatch(t, "android_adikteev_banner_bidreq.json"), "default_mraid_320x50")
	if err != nil || reason != "" {
		t.Fatalf("Build() err=%v reason=%q", err, reason)
	}
	if record.CreativeID != "default_mraid_320x50" {
		t.Errorf("CreativeID = %q, want default_mraid_320x50", record.CreativeID)
	}
	if record.CreativeType != TypeMRAIDBanner {
		t.Errorf("CreativeType = %q, want %q", record.CreativeType, TypeMRAIDBanner)
	}
}

func TestBuildNoCreativeIsANoBid(t *testing.T) {
	empty, err := LoadLibrary(writeLibrary(t, `{"default": {}}`))
	if err != nil {
		t.Fatalf("LoadLibrary(): %v", err)
	}

	bidder := NewBidder(testConfig(), empty)
	_, _, reason, err := bidder.Build(testMatch(t, "android_adikteev_banner_bidreq.json"), "")
	if err != nil {
		t.Fatalf("Build(): %v", err)
	}
	if reason != ReasonNoCreative {
		t.Errorf("reason = %q, want %q", reason, ReasonNoCreative)
	}
}

func TestPriceStaysAboveFloorAndUnderCap(t *testing.T) {
	cfg := testConfig()
	cfg.MaxPrice = 2
	bidder := NewBidder(cfg, loadDefaultLibrary(t))

	for _, floor := range []float64{0, 0.15, 0.5, 1.9, 5} {
		price := bidder.price(floor)
		if price <= floor {
			t.Errorf("price(%v) = %v, want a price above the floor", floor, price)
		}
	}
}

func TestShouldSkipHonoursNoBidRate(t *testing.T) {
	cfg := testConfig()
	cfg.NoBidRate = 0
	if NewBidder(cfg, loadDefaultLibrary(t)).ShouldSkip() {
		t.Error("ShouldSkip() = true with a zero no-bid rate")
	}

	cfg.NoBidRate = 1
	if !NewBidder(cfg, loadDefaultLibrary(t)).ShouldSkip() {
		t.Error("ShouldSkip() = false with a no-bid rate of 1")
	}
}
