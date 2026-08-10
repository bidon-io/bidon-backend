package adapters_builder_test

import (
	"context"
	"testing"

	"github.com/bidon-io/bidon-backend/config"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/auction"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters_builder"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

type stubConfigFetcher struct {
	profiles adapter.RawConfigsMap
	err      error
}

func (s stubConfigFetcher) FetchCached(context.Context, int64, []adapter.Key) (adapter.RawConfigsMap, error) {
	return s.profiles, s.err
}

func TestAdaptersConfigBuilder_Build_UsesRegistryRemaps(t *testing.T) {
	t.Parallel()

	fetcher := stubConfigFetcher{
		profiles: adapter.RawConfigsMap{
			adapter.MintegralKey: {
				AccountExtra: map[string]any{"publisher_id": "pub-1"},
				AppData:      map[string]any{"app_id": "app-1"},
			},
			adapter.MetaKey: {
				AccountExtra: map[string]any{},
				AppData:      map[string]any{"app_id": "meta-app"},
			},
			adapter.AdmobKey: {
				AccountExtra: map[string]any{"foo": "bar"},
				AppData:      map[string]any{},
			},
		},
	}

	adUnits := &auction.AdUnitsMap{
		adapter.MintegralKey: {
			{DemandID: string(adapter.MintegralKey), BidType: schema.RTBBidType, Extra: map[string]any{
				"unit_id":      "unit-1",
				"placement_id": "plc-1",
			}},
		},
		adapter.MetaKey: {
			{DemandID: string(adapter.MetaKey), BidType: schema.RTBBidType, Extra: map[string]any{
				"placement_id": "meta-plc",
			}},
		},
	}

	builder := adapters_builder.NewAdaptersConfigBuilder(fetcher, &config.DemandConfig{
		MetaAppSecret:  "secret",
		MetaPlatformID: "platform",
		MolocoAPIKey:   "moloco-key",
	})

	got, err := builder.Build(context.Background(), 1, []adapter.Key{
		adapter.MintegralKey,
		adapter.MetaKey,
		adapter.AdmobKey,
	}, adUnits)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	mintegral := got[adapter.MintegralKey]
	if mintegral["seller_id"] != "pub-1" || mintegral["app_id"] != "app-1" ||
		mintegral["tag_id"] != "unit-1" || mintegral["placement_id"] != "plc-1" {
		t.Fatalf("mintegral config = %#v", mintegral)
	}

	meta := got[adapter.MetaKey]
	if meta["app_id"] != "meta-app" || meta["tag_id"] != "meta-plc" ||
		meta["app_secret"] != "secret" || meta["platform_id"] != "platform" {
		t.Fatalf("meta config = %#v", meta)
	}

	admob := got[adapter.AdmobKey]
	if admob["foo"] != "bar" {
		t.Fatalf("admob passthrough = %#v, want account extra", admob)
	}
}

func TestAdaptersConfigBuilder_Build_SkipsAdUnitMapsWhenMissing(t *testing.T) {
	t.Parallel()

	fetcher := stubConfigFetcher{
		profiles: adapter.RawConfigsMap{
			adapter.VungleKey: {
				AccountExtra: map[string]any{"account_id": "acct"},
				AppData:      map[string]any{"app_id": "app"},
			},
		},
	}

	builder := adapters_builder.NewAdaptersConfigBuilder(fetcher, &config.DemandConfig{})
	got, err := builder.Build(context.Background(), 1, []adapter.Key{adapter.VungleKey}, &auction.AdUnitsMap{})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	cfg := got[adapter.VungleKey]
	if cfg["seller_id"] != "acct" || cfg["app_id"] != "app" {
		t.Fatalf("vungle config = %#v", cfg)
	}
	if _, ok := cfg["tag_id"]; ok {
		t.Fatalf("tag_id should be absent without RTB ad unit: %#v", cfg)
	}
}
