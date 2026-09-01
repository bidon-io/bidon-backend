package dspsim

import (
	"testing"

	"github.com/lib/pq"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/db"
)

// testAuctions and testLineItems mirror the shape of the sample seed data:
// one app with a banner, an interstitial and a rewarded auction, all bidding
// through adikteev.
func testAuctions() []auctionRow {
	return []auctionRow{
		{
			ID: 1, AppID: 10, AdType: db.BannerAdType, Pricefloor: 0.1,
			Bidding: pq.StringArray{"adikteev"}, PackageName: "com.demo.tetris", PlatformID: db.AndroidPlatformID,
		},
		{
			ID: 2, AppID: 10, AdType: db.InterstitialAdType, Pricefloor: 0.5,
			Bidding: pq.StringArray{"adikteev"}, PackageName: "com.demo.tetris", PlatformID: db.AndroidPlatformID,
		},
		{
			ID: 3, AppID: 10, AdType: db.RewardedAdType, Pricefloor: 0.5,
			Bidding: pq.StringArray{"adikteev"}, PackageName: "com.demo.tetris", PlatformID: db.AndroidPlatformID,
		},
	}
}

func testLineItems() []lineItemRow {
	return []lineItemRow{
		{ID: 100, AppID: 10, AdType: db.BannerAdType, Format: "BANNER", Width: 320, Height: 50, PublicUID: 900, HumanName: "banner", APIKey: "adikteev"},
		{ID: 101, AppID: 10, AdType: db.BannerAdType, Format: "MREC", Width: 300, Height: 250, PublicUID: 901, HumanName: "mrec", APIKey: "adikteev"},
		{ID: 102, AppID: 10, AdType: db.InterstitialAdType, PublicUID: 902, HumanName: "interstitial", APIKey: "adikteev"},
		{ID: 103, AppID: 10, AdType: db.RewardedAdType, PublicUID: 903, HumanName: "rewarded", APIKey: "adikteev"},
	}
}

func testCatalogStore() *CatalogStore {
	store := &CatalogStore{}
	store.current.Store(buildCatalog(testAuctions(), testLineItems()))
	return store
}

func TestBuildCatalog(t *testing.T) {
	catalog := buildCatalog(testAuctions(), testLineItems())

	if got := len(catalog.Auctions()); got != 3 {
		t.Fatalf("Auctions() = %d, want 3", got)
	}
	if !catalog.KnowsBundle("COM.DEMO.TETRIS") {
		t.Error("KnowsBundle() should be case-insensitive")
	}
	if catalog.KnowsBundle("com.unknown.app") {
		t.Error("KnowsBundle() returned true for an unconfigured bundle")
	}

	banner := catalog.Lookup("com.demo.tetris", ad.BannerType)
	if len(banner) != 1 {
		t.Fatalf("Lookup(banner) returned %d auctions, want 1", len(banner))
	}
	if len(banner[0].LineItems) != 2 {
		t.Errorf("banner auction has %d line items, want 2", len(banner[0].LineItems))
	}
	if banner[0].Platform != "android" {
		t.Errorf("Platform = %q, want android", banner[0].Platform)
	}
	if banner[0].LineItems[0].UID != "900" {
		t.Errorf("line item UID = %q, want 900", banner[0].LineItems[0].UID)
	}
}

func TestBuildCatalogFiltersByAdUnitIDs(t *testing.T) {
	auctions := testAuctions()
	auctions[0].AdUnitIds = pq.Int64Array{101} // MREC only

	catalog := buildCatalog(auctions, testLineItems())
	banner := catalog.Lookup("com.demo.tetris", ad.BannerType)[0]

	if len(banner.LineItems) != 1 || banner.LineItems[0].ID != 101 {
		t.Fatalf("ad_unit_ids not honoured, got %+v", banner.LineItems)
	}
	if banner.SupportsFormat(ad.BannerFormat) {
		t.Error("SupportsFormat(BANNER) = true after filtering down to the MREC ad unit")
	}
	if !banner.SupportsFormat(ad.MRECFormat) {
		t.Error("SupportsFormat(MREC) = false")
	}
}

func TestConfiguredAuctionSupportsFormat(t *testing.T) {
	auction := &ConfiguredAuction{AdType: ad.BannerType, Formats: []ad.Format{ad.AdaptiveFormat}}

	tests := []struct {
		format ad.Format
		want   bool
	}{
		{ad.BannerFormat, true},      // ADAPTIVE covers BANNER
		{ad.LeaderboardFormat, true}, // and LEADERBOARD
		{ad.MRECFormat, false},       // but never MREC
		{ad.AdaptiveFormat, true},
	}
	for _, tt := range tests {
		if got := auction.SupportsFormat(tt.format); got != tt.want {
			t.Errorf("SupportsFormat(%s) = %v, want %v", tt.format, got, tt.want)
		}
	}

	fullscreen := &ConfiguredAuction{AdType: ad.InterstitialType}
	if !fullscreen.SupportsFormat(ad.EmptyFormat) {
		t.Error("fullscreen auctions should accept an empty format")
	}
}

func TestConfiguredAuctionHasDemand(t *testing.T) {
	auction := &ConfiguredAuction{
		Demands:   []string{"Adikteev"},
		LineItems: []CatalogLineItem{{Demand: "bidmachine"}},
	}

	for _, key := range []string{"adikteev", "ADIKTEEV", "bidmachine"} {
		if !auction.HasDemand(key) {
			t.Errorf("HasDemand(%q) = false", key)
		}
	}
	if auction.HasDemand("meta") {
		t.Error("HasDemand(meta) = true")
	}
}
