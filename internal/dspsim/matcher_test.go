package dspsim

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/ad"
)

// fixtureDir holds real OpenRTB bid requests captured from the Adikteev
// adapter, which is exactly what the simulator has to understand.
const fixtureDir = "../sdkapi/v2/apihandlers/testdata/auction/adikteev"

func loadBidRequest(t *testing.T, name string) *openrtb2.BidRequest {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(fixtureDir, name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}

	var request openrtb2.BidRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("unmarshal fixture %s: %v", name, err)
	}
	return &request
}

func TestDescribeCapturedRequests(t *testing.T) {
	tests := []struct {
		fixture    string
		wantFormat ad.Format
		fullscreen bool
	}{
		{"android_adikteev_banner_bidreq.json", ad.BannerFormat, false},
		{"ios_adikteev_banner_bidreq.json", ad.BannerFormat, false},
		{"android_adikteev_banner_mrec_bidreq.json", ad.MRECFormat, false},
		{"ios_adikteev_banner_mrec_bidreq.json", ad.MRECFormat, false},
		{"android_adikteev_interstitial_bidreq.json", ad.EmptyFormat, true},
		{"ios_adikteev_interstitial_bidreq.json", ad.EmptyFormat, true},
		{"android_adikteev_rewarded_bidreq.json", ad.EmptyFormat, true},
		{"ios_adikteev_rewarded_bidreq.json", ad.EmptyFormat, true},
	}

	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			summary, err := Describe(loadBidRequest(t, tt.fixture))
			if err != nil {
				t.Fatalf("Describe() error: %v", err)
			}

			if summary.Bundle != "com.demo.tetris" {
				t.Errorf("Bundle = %q, want com.demo.tetris", summary.Bundle)
			}
			if summary.DSP != "adikteev" {
				t.Errorf("DSP = %q, want adikteev", summary.DSP)
			}
			if summary.Format != tt.wantFormat {
				t.Errorf("Format = %q, want %q", summary.Format, tt.wantFormat)
			}
			if summary.Fullscreen != tt.fullscreen {
				t.Errorf("Fullscreen = %v, want %v", summary.Fullscreen, tt.fullscreen)
			}
			if summary.Floor <= 0 {
				t.Errorf("Floor = %v, want the impression floor", summary.Floor)
			}
			if summary.ImpID == "" {
				t.Error("ImpID is empty")
			}
		})
	}
}

func TestDescribeRejectsRequestWithoutImpression(t *testing.T) {
	if _, err := Describe(&openrtb2.BidRequest{ID: "x"}); err != ErrNoImpression {
		t.Fatalf("Describe() error = %v, want ErrNoImpression", err)
	}
	if _, err := Describe(nil); err != ErrNoImpression {
		t.Fatalf("Describe(nil) error = %v, want ErrNoImpression", err)
	}
}

func TestMatchCapturedRequests(t *testing.T) {
	matcher := &Matcher{Catalog: testCatalogStore(), MaxPrice: 25}

	tests := []struct {
		fixture string
		wantAd  ad.Type
	}{
		{"android_adikteev_banner_bidreq.json", ad.BannerType},
		{"android_adikteev_banner_mrec_bidreq.json", ad.BannerType},
		{"android_adikteev_interstitial_bidreq.json", ad.InterstitialType},
		// Video-only fullscreen impressions resolve to rewarded when the app
		// has a rewarded auction configured.
		{"android_adikteev_rewarded_bidreq.json", ad.RewardedType},
	}

	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			summary, err := Describe(loadBidRequest(t, tt.fixture))
			if err != nil {
				t.Fatalf("Describe() error: %v", err)
			}

			match, reason := matcher.Match(summary, ad.UnknownType)
			if reason != "" {
				t.Fatalf("Match() no-bid reason = %q, want a match", reason)
			}
			if match.Summary.AdType != tt.wantAd {
				t.Errorf("AdType = %q, want %q", match.Summary.AdType, tt.wantAd)
			}
			if !match.DemandConfigured {
				t.Error("DemandConfigured = false, adikteev is configured as bidding demand")
			}
		})
	}
}

func TestMatchAdTypeOverride(t *testing.T) {
	matcher := &Matcher{Catalog: testCatalogStore(), MaxPrice: 25}

	summary, err := Describe(loadBidRequest(t, "android_adikteev_rewarded_bidreq.json"))
	if err != nil {
		t.Fatalf("Describe() error: %v", err)
	}

	match, reason := matcher.Match(summary, ad.InterstitialType)
	if reason != "" {
		t.Fatalf("Match() no-bid reason = %q", reason)
	}
	if match.Summary.AdType != ad.InterstitialType {
		t.Errorf("AdType = %q, want interstitial (override ignored)", match.Summary.AdType)
	}
}

func TestMatchNoBidReasons(t *testing.T) {
	matcher := &Matcher{Catalog: testCatalogStore(), MaxPrice: 25}

	base := RequestSummary{
		Bundle: "com.demo.tetris",
		DSP:    "adikteev",
		AdType: ad.BannerType,
		Format: ad.BannerFormat,
		Floor:  0.15,
	}

	t.Run("unknown bundle", func(t *testing.T) {
		summary := base
		summary.Bundle = "com.unknown.app"
		if _, reason := matcher.Match(summary, ad.UnknownType); reason != ReasonUnknownBundle {
			t.Errorf("reason = %q, want %q", reason, ReasonUnknownBundle)
		}
	})

	t.Run("floor above cap", func(t *testing.T) {
		summary := base
		summary.Floor = 1_000
		if _, reason := matcher.Match(summary, ad.UnknownType); reason != ReasonFloorTooHigh {
			t.Errorf("reason = %q, want %q", reason, ReasonFloorTooHigh)
		}
	})

	t.Run("format not configured", func(t *testing.T) {
		catalog := &CatalogStore{}
		auctions := testAuctions()
		lineItems := []lineItemRow{
			{ID: 100, AppID: 10, AdType: 3, Format: "MREC", Width: 300, Height: 250, APIKey: "adikteev"},
		}
		catalog.current.Store(buildCatalog(auctions[:1], lineItems))

		m := &Matcher{Catalog: catalog, MaxPrice: 25}
		if _, reason := m.Match(base, ad.UnknownType); reason != ReasonFormatNotConfigured {
			t.Errorf("reason = %q, want %q", reason, ReasonFormatNotConfigured)
		}
	})

	t.Run("no config for ad type", func(t *testing.T) {
		catalog := &CatalogStore{}
		catalog.current.Store(buildCatalog(testAuctions()[:1], testLineItems()))

		m := &Matcher{Catalog: catalog, MaxPrice: 25}
		summary := base
		summary.AdType = ad.InterstitialType
		summary.Format = ad.EmptyFormat
		summary.AdTypeAmbiguous = true

		if _, reason := m.Match(summary, ad.UnknownType); reason != ReasonNoConfigForAdType {
			t.Errorf("reason = %q, want %q", reason, ReasonNoConfigForAdType)
		}
	})
}

func TestMatchStrictDemand(t *testing.T) {
	summary := RequestSummary{
		Bundle: "com.demo.tetris",
		DSP:    "meta",
		AdType: ad.BannerType,
		Format: ad.BannerFormat,
		Floor:  0.15,
	}

	soft := &Matcher{Catalog: testCatalogStore(), MaxPrice: 25}
	match, reason := soft.Match(summary, ad.UnknownType)
	if reason != "" {
		t.Fatalf("soft matcher returned reason %q, want a match", reason)
	}
	if match.DemandConfigured {
		t.Error("DemandConfigured = true for a demand absent from the auction")
	}

	strict := &Matcher{Catalog: testCatalogStore(), MaxPrice: 25, StrictDemand: true}
	if _, reason := strict.Match(summary, ad.UnknownType); reason != ReasonDemandNotConfigured {
		t.Errorf("strict matcher reason = %q, want %q", reason, ReasonDemandNotConfigured)
	}
}

func TestParseAdType(t *testing.T) {
	tests := map[string]ad.Type{
		"banner":       ad.BannerType,
		"Interstitial": ad.InterstitialType,
		" rewarded ":   ad.RewardedType,
		"":             ad.UnknownType,
		"nonsense":     ad.UnknownType,
	}
	for in, want := range tests {
		if got := ParseAdType(in); got != want {
			t.Errorf("ParseAdType(%q) = %q, want %q", in, got, want)
		}
	}
}
