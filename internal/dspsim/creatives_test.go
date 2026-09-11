package dspsim

import (
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bidon-io/bidon-backend/internal/ad"
)

func testRand() *rand.Rand {
	return rand.New(rand.NewSource(1)) //nolint:gosec // deterministic test fixture
}

func loadDefaultLibrary(t *testing.T) *Library {
	t.Helper()

	lib, err := LoadLibrary("")
	if err != nil {
		t.Fatalf("LoadLibrary(embedded): %v", err)
	}
	return lib
}

func TestLoadDefaultLibrary(t *testing.T) {
	lib := loadDefaultLibrary(t)

	buckets := lib.Buckets()
	if len(buckets) < 2 {
		t.Fatalf("Buckets() = %v, want at least default and one DSP bucket", buckets)
	}
	if lib.Source != "embedded" {
		t.Errorf("Source = %q, want embedded", lib.Source)
	}

	for _, bucket := range buckets {
		for _, creative := range lib.Describe(bucket)[bucket] {
			if creative.Weight <= 0 {
				t.Errorf("creative %q has weight %d", creative.ID, creative.Weight)
			}
		}
	}
}

// The default library must be able to serve every slot the captured Adikteev
// requests ask for.
func TestDefaultLibraryCoversEveryCapturedSlot(t *testing.T) {
	lib := loadDefaultLibrary(t)

	slots := []struct {
		name   string
		adType ad.Type
		format ad.Format
		w, h   int64
	}{
		{"banner", ad.BannerType, ad.BannerFormat, 320, 50},
		{"mrec", ad.BannerType, ad.MRECFormat, 300, 250},
		{"interstitial", ad.InterstitialType, ad.EmptyFormat, 320, 480},
		{"rewarded", ad.RewardedType, ad.EmptyFormat, 360, 640},
	}

	for _, dsp := range []string{"adikteev", "unknown_dsp", ""} {
		for _, slot := range slots {
			t.Run(dsp+"/"+slot.name, func(t *testing.T) {
				selection, ok := lib.Select(dsp, slot.adType, slot.format, slot.w, slot.h, "", testRand())
				if !ok {
					t.Fatalf("Select(%q, %s) found no creative", dsp, slot.name)
				}

				adm, err := selection.Creative.Render(CreativeData{
					BidID: "bid-1", CreativeID: selection.Creative.ID, Price: 1.23,
					Currency: "USD", PublicURL: "http://sim",
					ClickURL: "http://sim/click", ImpressionURL: "http://sim/imp",
					TrackURL: "http://sim/track", AssetURL: "http://sim/asset",
				})
				if err != nil {
					t.Fatalf("Render(): %v", err)
				}
				if strings.Contains(adm, "{{") {
					t.Errorf("rendered adm still contains a template action: %s", adm)
				}
				if adm == "" {
					t.Error("rendered adm is empty")
				}
			})
		}
	}
}

func TestSelectRoutesByDSPAndFallsBack(t *testing.T) {
	lib := loadDefaultLibrary(t)

	t.Run("known dsp uses its own bucket", func(t *testing.T) {
		selection, ok := lib.Select("adikteev", ad.BannerType, ad.BannerFormat, 320, 50, "", testRand())
		if !ok {
			t.Fatal("Select() found no creative")
		}
		if selection.Bucket != "adikteev" {
			t.Errorf("Bucket = %q, want adikteev", selection.Bucket)
		}
		if selection.FellBack {
			t.Error("FellBack = true for a DSP with its own creatives")
		}
	})

	t.Run("unknown dsp falls back to default", func(t *testing.T) {
		selection, ok := lib.Select("does_not_exist", ad.BannerType, ad.BannerFormat, 320, 50, "", testRand())
		if !ok {
			t.Fatal("Select() found no creative")
		}
		if selection.Bucket != DefaultDSPBucket || !selection.FellBack {
			t.Errorf("Bucket = %q FellBack = %v, want default/true", selection.Bucket, selection.FellBack)
		}
	})

	t.Run("empty display manager uses default", func(t *testing.T) {
		selection, ok := lib.Select("", ad.BannerType, ad.BannerFormat, 320, 50, "", testRand())
		if !ok {
			t.Fatal("Select() found no creative")
		}
		if selection.Bucket != DefaultDSPBucket {
			t.Errorf("Bucket = %q, want default", selection.Bucket)
		}
	})
}

// A DSP bucket that cannot serve the requested format falls back to default
// rather than no-bidding.
func TestSelectFallsBackWhenBucketLacksFormat(t *testing.T) {
	path := writeLibrary(t, `{
		"default": {
			"static_banner": [
				{"id": "d_banner", "w": 320, "h": 50, "formats": ["BANNER"], "adm": "<a href=\"{{.ClickURL}}\">d</a>"},
				{"id": "d_mrec", "w": 300, "h": 250, "formats": ["MREC"], "adm": "<a href=\"{{.ClickURL}}\">d</a>"}
			]
		},
		"partial": {
			"static_banner": [
				{"id": "p_banner", "w": 320, "h": 50, "formats": ["BANNER"], "adm": "<a href=\"{{.ClickURL}}\">p</a>"}
			]
		}
	}`)

	lib, err := LoadLibrary(path)
	if err != nil {
		t.Fatalf("LoadLibrary(): %v", err)
	}

	selection, ok := lib.Select("partial", ad.BannerType, ad.BannerFormat, 320, 50, "", testRand())
	if !ok || selection.Bucket != "partial" {
		t.Fatalf("BANNER should be served from the partial bucket, got %+v ok=%v", selection, ok)
	}

	selection, ok = lib.Select("partial", ad.BannerType, ad.MRECFormat, 300, 250, "", testRand())
	if !ok {
		t.Fatal("MREC should fall back to the default bucket")
	}
	if selection.Bucket != DefaultDSPBucket || !selection.FellBack || selection.Creative.ID != "d_mrec" {
		t.Errorf("got bucket=%q fellBack=%v id=%q, want default/true/d_mrec",
			selection.Bucket, selection.FellBack, selection.Creative.ID)
	}
}

func TestSelectForcedCreative(t *testing.T) {
	lib := loadDefaultLibrary(t)

	selection, ok := lib.Select("adikteev", ad.BannerType, ad.BannerFormat, 320, 50, "default_static_320x50", testRand())
	if !ok {
		t.Fatal("forcing a creative from the default bucket should succeed")
	}
	if selection.Creative.ID != "default_static_320x50" || selection.Bucket != DefaultDSPBucket {
		t.Errorf("got id=%q bucket=%q", selection.Creative.ID, selection.Bucket)
	}

	if _, ok := lib.Select("adikteev", ad.BannerType, ad.BannerFormat, 320, 50, "nope", testRand()); ok {
		t.Error("forcing an unknown creative id should fail")
	}
}

func TestSelectIsDeterministicUnderAFixedSeed(t *testing.T) {
	lib := loadDefaultLibrary(t)

	var first []string
	for range 10 {
		selection, ok := lib.Select("default", ad.BannerType, ad.BannerFormat, 320, 50, "", testRand())
		if !ok {
			t.Fatal("Select() found no creative")
		}
		first = append(first, selection.Creative.ID)
	}

	for i := 1; i < len(first); i++ {
		if first[i] != first[0] {
			t.Fatalf("same seed produced different creatives: %v", first)
		}
	}
}

func TestSelectRespectsWeights(t *testing.T) {
	path := writeLibrary(t, `{
		"default": {
			"static_banner": [
				{"id": "heavy", "w": 320, "h": 50, "formats": ["BANNER"], "weight": 99, "adm": "<a>h</a>"},
				{"id": "light", "w": 320, "h": 50, "formats": ["BANNER"], "weight": 1, "adm": "<a>l</a>"}
			]
		}
	}`)

	lib, err := LoadLibrary(path)
	if err != nil {
		t.Fatalf("LoadLibrary(): %v", err)
	}

	rnd := testRand()
	counts := map[string]int{}
	for range 1_000 {
		selection, ok := lib.Select("default", ad.BannerType, ad.BannerFormat, 320, 50, "", rnd)
		if !ok {
			t.Fatal("Select() found no creative")
		}
		counts[selection.Creative.ID]++
	}

	if counts["heavy"] <= counts["light"]*10 {
		t.Errorf("weighting not applied: %v", counts)
	}
}

func TestLoadLibraryValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "missing default bucket",
			content: `{"adikteev": {"static_banner": [{"id": "a", "formats": ["BANNER"], "adm": "<a></a>"}]}}`,
			wantErr: `missing "default" bucket`,
		},
		{
			name: "duplicate id within a bucket",
			content: `{"default": {
				"static_banner": [{"id": "dup", "formats": ["BANNER"], "adm": "<a></a>"}],
				"mraid_banner": [{"id": "dup", "formats": ["BANNER"], "adm": "<a></a>"}]
			}}`,
			wantErr: "duplicate creative id",
		},
		{
			name:    "unparseable template",
			content: `{"default": {"static_banner": [{"id": "bad", "formats": ["BANNER"], "adm": "{{.Click"}]}}`,
			wantErr: "invalid adm template",
		},
		{
			name:    "unknown format",
			content: `{"default": {"static_banner": [{"id": "bad", "formats": ["HUGE"], "adm": "<a></a>"}]}}`,
			wantErr: "unknown format",
		},
		{
			name:    "unknown ad type",
			content: `{"default": {"vast_video": [{"id": "bad", "ad_types": ["playable"], "adm": "<VAST/>"}]}}`,
			wantErr: "unknown ad type",
		},
		{
			name:    "empty adm",
			content: `{"default": {"static_banner": [{"id": "bad", "formats": ["BANNER"], "adm": "  "}]}}`,
			wantErr: "empty adm",
		},
		{
			name:    "no formats and no ad types",
			content: `{"default": {"static_banner": [{"id": "bad", "adm": "<a></a>"}]}}`,
			wantErr: "neither formats nor ad_types",
		},
		{
			name:    "missing id",
			content: `{"default": {"static_banner": [{"formats": ["BANNER"], "adm": "<a></a>"}]}}`,
			wantErr: "has no id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadLibrary(writeLibrary(t, tt.content))
			if err == nil {
				t.Fatalf("LoadLibrary() succeeded, want error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("LoadLibrary() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadLibraryMissingFile(t *testing.T) {
	if _, err := LoadLibrary(filepath.Join(t.TempDir(), "absent.json")); err == nil {
		t.Fatal("LoadLibrary() on a missing file should fail")
	}
}

func TestMarkServedIsReportedByDescribe(t *testing.T) {
	lib := loadDefaultLibrary(t)
	lib.MarkServed(DefaultDSPBucket, "default_static_320x50")
	lib.MarkServed(DefaultDSPBucket, "default_static_320x50")

	for _, summary := range lib.Describe(DefaultDSPBucket)[DefaultDSPBucket] {
		if summary.ID == "default_static_320x50" && summary.Served != 2 {
			t.Errorf("Served = %d, want 2", summary.Served)
		}
	}
}

func writeLibrary(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "creatives.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write library: %v", err)
	}
	return path
}
