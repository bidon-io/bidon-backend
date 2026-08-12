package adapters_builder_test

import (
	"context"
	"reflect"
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

// Remap expectations mirror the pre-registry AdaptersConfigBuilder switch so
// registry FieldMap / InjectEnvSecrets migrations stay behaviorally locked.
func TestAdaptersConfigBuilder_Build_RemapParity(t *testing.T) {
	t.Parallel()

	demandCfg := &config.DemandConfig{
		MetaAppSecret:  "meta-secret",
		MetaPlatformID: "meta-platform",
		MolocoAPIKey:   "moloco-api-key",
	}

	tests := []struct {
		name         string
		key          adapter.Key
		accountExtra map[string]any
		appData      map[string]any
		adUnitExtra  map[string]any // nil => no RTB ad unit
		want         map[string]any
	}{
		{
			name:         "adikteev",
			key:          adapter.AdikteevKey,
			accountExtra: map[string]any{"sdk_instance_id": "sdk-1"},
			appData:      map[string]any{},
			want:         map[string]any{"sdk_instance_id": "sdk-1"},
		},
		{
			name:         "amazon",
			key:          adapter.AmazonKey,
			accountExtra: map[string]any{"price_points_map": map[string]any{"a": 1.0}},
			appData:      map[string]any{},
			want:         map[string]any{"price_points_map": map[string]any{"a": 1.0}},
		},
		{
			name: "bidmachine",
			key:  adapter.BidmachineKey,
			accountExtra: map[string]any{
				"seller_id":        "seller",
				"endpoint":         "https://example.test",
				"mediation_config": []any{"m1"},
			},
			appData: map[string]any{},
			want: map[string]any{
				"seller_id":        "seller",
				"endpoint":         "https://example.test",
				"mediation_config": []any{"m1"},
			},
		},
		{
			name:         "bigoads",
			key:          adapter.BigoAdsKey,
			accountExtra: map[string]any{"publisher_id": "pub"},
			appData:      map[string]any{"app_id": "app"},
			adUnitExtra:  map[string]any{"slot_id": "slot", "placement_id": "plc"},
			want: map[string]any{
				"seller_id":    "pub",
				"app_id":       "app",
				"tag_id":       "slot",
				"placement_id": "plc",
			},
		},
		{
			name:         "inmobi",
			key:          adapter.InmobiKey,
			accountExtra: map[string]any{},
			appData:      map[string]any{"app_key": "inmobi-key"},
			adUnitExtra:  map[string]any{"placement_id": "plc"},
			want: map[string]any{
				"app_id":       "inmobi-key",
				"placement_id": "plc",
			},
		},
		{
			name:         "mintegral",
			key:          adapter.MintegralKey,
			accountExtra: map[string]any{"publisher_id": "pub"},
			appData:      map[string]any{"app_id": "app"},
			adUnitExtra:  map[string]any{"unit_id": "unit", "placement_id": "plc"},
			want: map[string]any{
				"seller_id":    "pub",
				"app_id":       "app",
				"tag_id":       "unit",
				"placement_id": "plc",
			},
		},
		{
			name:         "vkads",
			key:          adapter.VKAdsKey,
			accountExtra: map[string]any{},
			appData:      map[string]any{"app_id": "app"},
			adUnitExtra:  map[string]any{"slot_id": "slot"},
			want: map[string]any{
				"app_id": "app",
				"tag_id": "slot",
			},
		},
		{
			name:         "vungle",
			key:          adapter.VungleKey,
			accountExtra: map[string]any{"account_id": "acct"},
			appData:      map[string]any{"app_id": "app"},
			adUnitExtra:  map[string]any{"placement_id": "plc"},
			want: map[string]any{
				"seller_id": "acct",
				"app_id":    "app",
				"tag_id":    "plc",
			},
		},
		{
			name:         "meta",
			key:          adapter.MetaKey,
			accountExtra: map[string]any{},
			appData:      map[string]any{"app_id": "meta-app"},
			adUnitExtra:  map[string]any{"placement_id": "plc"},
			want: map[string]any{
				"app_id":      "meta-app",
				"app_secret":  "meta-secret",
				"platform_id": "meta-platform",
				"tag_id":      "plc",
			},
		},
		{
			name:         "mobilefuse",
			key:          adapter.MobileFuseKey,
			accountExtra: map[string]any{},
			appData:      map[string]any{},
			adUnitExtra:  map[string]any{"placement_id": "plc"},
			want:         map[string]any{"tag_id": "plc"},
		},
		{
			name:         "moloco",
			key:          adapter.MolocoKey,
			accountExtra: map[string]any{},
			appData:      map[string]any{"app_key": "moloco-key"},
			adUnitExtra:  map[string]any{"ad_unit_id": "au"},
			want: map[string]any{
				"app_id":  "moloco-key",
				"api_key": "moloco-api-key",
				"tag_id":  "au",
			},
		},
		{
			name:         "startio",
			key:          adapter.StartIOKey,
			accountExtra: map[string]any{"account": "acct"},
			appData:      map[string]any{"app_id": "app"},
			adUnitExtra:  map[string]any{"tag_id": "tag"},
			want: map[string]any{
				"account": "acct",
				"app_id":  "app",
				"tag_id":  "tag",
			},
		},
		{
			name:         "taurusx",
			key:          adapter.TaurusXKey,
			accountExtra: map[string]any{},
			appData:      map[string]any{"app_id": "app"},
			adUnitExtra:  map[string]any{"placement_id": "plc"},
			want: map[string]any{
				"app_id": "app",
				"tag_id": "plc",
			},
		},
		{
			name:         "yandex",
			key:          adapter.YandexKey,
			accountExtra: map[string]any{},
			appData:      map[string]any{},
			adUnitExtra:  map[string]any{"ad_unit_id": "au"},
			want:         map[string]any{"ad_unit_id": "au"},
		},
		{
			name:         "zmaticoo",
			key:          adapter.ZmaticooKey,
			accountExtra: map[string]any{},
			appData:      map[string]any{"app_key": "z-key"},
			adUnitExtra:  map[string]any{"placement_id": "plc"},
			want: map[string]any{
				"app_id":       "z-key",
				"placement_id": "plc",
			},
		},
		{
			name:         "admob_passthrough",
			key:          adapter.AdmobKey,
			accountExtra: map[string]any{"foo": "bar", "nested": map[string]any{"k": "v"}},
			appData:      map[string]any{"ignored": true},
			want:         map[string]any{"foo": "bar", "nested": map[string]any{"k": "v"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fetcher := stubConfigFetcher{
				profiles: adapter.RawConfigsMap{
					tt.key: {
						AccountExtra: tt.accountExtra,
						AppData:      tt.appData,
					},
				},
			}

			var adUnits *auction.AdUnitsMap
			if tt.adUnitExtra != nil {
				adUnits = &auction.AdUnitsMap{
					tt.key: {
						{
							DemandID: string(tt.key),
							BidType:  schema.RTBBidType,
							Extra:    tt.adUnitExtra,
						},
					},
				}
			} else {
				adUnits = &auction.AdUnitsMap{}
			}

			builder := adapters_builder.NewAdaptersConfigBuilder(fetcher, demandCfg)
			got, err := builder.Build(context.Background(), 1, []adapter.Key{tt.key}, adUnits)
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}

			if !reflect.DeepEqual(got[tt.key], tt.want) {
				t.Fatalf("processed config mismatch\ngot:  %#v\nwant: %#v", got[tt.key], tt.want)
			}
		})
	}
}
