package rendering_test

import (
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/rendering"
)

const testDemandID = adapter.Key("test-dsp")

func TestValidate_validConfig(t *testing.T) {
	t.Parallel()

	delay := 5.0
	enabled := true
	cfg := rendering.Config{
		CloseButton: &rendering.CloseButtonConfig{
			Style:        rendering.CloseButtonStyleIconX,
			Position:     rendering.CloseButtonPositionTopRight,
			Color:        "#FFFFFF",
			DelaySeconds: &delay,
		},
		Creative: &rendering.CreativeConfig{
			Type: rendering.CreativeTypeVAST,
		},
		StoreKit: &rendering.StoreKitConfig{
			Enabled:    &enabled,
			AppStoreID: "123456789",
			Trigger:    rendering.StoreKitTriggerOnEndCard,
		},
	}

	err := rendering.Validate(&cfg)
	assert.NoError(t, err)
}

func TestValidate_emptyConfig(t *testing.T) {
	t.Parallel()

	err := rendering.Validate(&rendering.Config{})
	assert.NoError(t, err)
}

func TestValidate_errors(t *testing.T) {
	t.Parallel()

	opacity := 1.5
	enabled := true

	tests := []struct {
		name  string
		cfg   rendering.Config
		field string
		tag   string
	}{
		{
			name: "invalid close button style",
			cfg: rendering.Config{
				CloseButton: &rendering.CloseButtonConfig{Style: rendering.CloseButtonStyle("invalid")},
			},
			field: "CloseButton.Style",
			tag:   "oneof",
		},
		{
			name: "custom style without asset url",
			cfg: rendering.Config{
				CloseButton: &rendering.CloseButtonConfig{Style: rendering.CloseButtonStyleCustom},
			},
			field: "CloseButton.CustomAssetURL",
			tag:   "custom_asset_url",
		},
		{
			name: "invalid opacity",
			cfg: rendering.Config{
				CloseButton: &rendering.CloseButtonConfig{Opacity: &opacity},
			},
			field: "CloseButton.Opacity",
			tag:   "lte",
		},
		{
			name: "invalid endcard count",
			cfg: rendering.Config{
				EndCards: &rendering.EndCardsConfig{Count: intPtr(4)},
			},
			field: "EndCards.Count",
			tag:   "lte",
		},
		{
			name: "creative type required",
			cfg: rendering.Config{
				Creative: &rendering.CreativeConfig{},
			},
			field: "Creative.Type",
			tag:   "required",
		},
		{
			name: "invalid creative type",
			cfg: rendering.Config{
				Creative: &rendering.CreativeConfig{Type: rendering.CreativeType("flash")},
			},
			field: "Creative.Type",
			tag:   "oneof",
		},
		{
			name: "store kit enabled without app id",
			cfg: rendering.Config{
				StoreKit: &rendering.StoreKitConfig{Enabled: &enabled},
			},
			field: "StoreKit.AppStoreID",
			tag:   "required_if",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := rendering.Validate(&tt.cfg)
			require.Error(t, err)
			errMsg := err.Error()
			assert.Contains(t, errMsg, tt.field)
			assert.Contains(t, errMsg, tt.tag)
		})
	}
}

func TestValidate_collectsMultipleErrors(t *testing.T) {
	t.Parallel()

	opacity := 1.5
	enabled := true
	cfg := rendering.Config{
		CloseButton: &rendering.CloseButtonConfig{
			Style:   rendering.CloseButtonStyle("invalid"),
			Opacity: &opacity,
		},
		EndCards: &rendering.EndCardsConfig{Count: intPtr(4)},
		Creative: &rendering.CreativeConfig{Type: rendering.CreativeType("flash")},
		StoreKit: &rendering.StoreKitConfig{Enabled: &enabled},
	}

	err := rendering.Validate(&cfg)
	require.Error(t, err)

	errMsg := err.Error()
	assert.Contains(t, errMsg, "CloseButton.Style")
	assert.Contains(t, errMsg, "CloseButton.Opacity")
	assert.Contains(t, errMsg, "EndCards.Count")
	assert.Contains(t, errMsg, "Creative.Type")
	assert.Contains(t, errMsg, "StoreKit.AppStoreID")
}

func TestApplyDefaults_fillsMissingFields(t *testing.T) {
	t.Parallel()

	cfg := &rendering.Config{
		Container: &rendering.ContainerConfig{
			Format: rendering.ContainerFormatInterstitial,
		},
	}
	require.NoError(t, rendering.ApplyDefaults(cfg))
	require.NotNil(t, cfg.Container)
	assert.Equal(t, rendering.ContainerOrientationResponsive, cfg.Container.Orientation)

	// A section the caller never mentioned is backfilled with its own defaults too.
	require.NotNil(t, cfg.CloseButton)
	assert.Equal(t, rendering.CloseButtonStyleIconX, cfg.CloseButton.Style)
}

func TestParseFromBidExt_validRendering(t *testing.T) {
	t.Parallel()

	bidExt := []byte(`{
		"signaldata": "abc",
		"rendering": {
			"creative": { "type": "vast" },
			"container": { "format": "interstitial" }
		}
	}`)

	got := rendering.ParseFromBidExt(bidExt, testDemandID)
	require.NotNil(t, got)
	require.NotNil(t, got.Creative)
	assert.Equal(t, rendering.CreativeTypeVAST, got.Creative.Type)
	assert.Equal(t, rendering.CreativeVASTVersionV42, got.Creative.VASTVersion)
	require.NotNil(t, got.Container)
	assert.Equal(t, rendering.ContainerFormatInterstitial, got.Container.Format)
	assert.Equal(t, rendering.ContainerOrientationResponsive, got.Container.Orientation)

	// close_button was never mentioned in the payload but is still backfilled with defaults.
	require.NotNil(t, got.CloseButton)
	assert.Equal(t, rendering.CloseButtonStyleIconX, got.CloseButton.Style)
}

// A malformed section falls back to that section's own defaults. Sections the DSP never
// mentioned (close_button here) are backfilled with defaults too, same as if they'd been
// sent as "{}" - omission and an empty object behave identically.
func TestParseFromBidExt_invalidRenderingFallsBackToDefaults(t *testing.T) {
	t.Parallel()

	bidExt := []byte(`{
		"rendering": {
			"creative": { "type": "unsupported" }
		}
	}`)

	got := rendering.ParseFromBidExt(bidExt, testDemandID)
	require.NotNil(t, got)
	require.NotNil(t, got.CloseButton)
	assert.Equal(t, rendering.CloseButtonStyleIconX, got.CloseButton.Style)
	require.NotNil(t, got.Creative)
	assert.Equal(t, rendering.CreativeTypeStaticImage, got.Creative.Type)
}

// An invalid section falls back to defaults without discarding a sibling section that did
// validate successfully.
func TestParseFromBidExt_invalidSectionPreservesValidSections(t *testing.T) {
	t.Parallel()

	bidExt := []byte(`{
		"rendering": {
			"close_button": { "style": "icon_circle", "color": "#112233" },
			"creative": { "type": "unsupported" }
		}
	}`)

	got := rendering.ParseFromBidExt(bidExt, testDemandID)
	require.NotNil(t, got)

	require.NotNil(t, got.CloseButton)
	assert.Equal(t, rendering.CloseButtonStyleIconCircle, got.CloseButton.Style)
	assert.Equal(t, "#112233", got.CloseButton.Color)

	require.NotNil(t, got.Creative)
	assert.Equal(t, rendering.CreativeTypeStaticImage, got.Creative.Type)
}

// StoreKit.AppStoreID's requirement is connected to StoreKit.Enabled (required_if). Sending
// enabled without app_store_id doesn't just leave app_store_id blank - the whole section,
// including Enabled itself, resets to defaults, because "enabled: true, app_store_id: unset"
// was never a valid combination to partially preserve.
func TestParseFromBidExt_connectedFieldFailureResetsWholeSection(t *testing.T) {
	t.Parallel()

	bidExt := []byte(`{
		"rendering": {
			"store_kit": { "enabled": true }
		}
	}`)

	got := rendering.ParseFromBidExt(bidExt, testDemandID)
	require.NotNil(t, got)
	require.NotNil(t, got.StoreKit)
	require.NotNil(t, got.StoreKit.Enabled)
	assert.False(t, *got.StoreKit.Enabled)
	assert.Empty(t, got.StoreKit.AppStoreID)
}

// Omitting rendering entirely, sending "rendering": {}, and omitting one specific section
// all backfill defaults the same way - there is no separate opt-in step.
func TestParseFromBidExt_omissionEmptyObjectAndAbsentRenderingAreEquivalent(t *testing.T) {
	t.Parallel()

	omittedRendering := rendering.ParseFromBidExt([]byte(`{}`), testDemandID)
	emptyRendering := rendering.ParseFromBidExt([]byte(`{"rendering": {}}`), testDemandID)
	omittedSection := rendering.ParseFromBidExt([]byte(`{"rendering": {"endcards": {}}}`), testDemandID)

	require.NotNil(t, omittedRendering.CloseButton)
	require.NotNil(t, emptyRendering.CloseButton)
	require.NotNil(t, omittedSection.CloseButton)
	assert.Equal(t, omittedRendering.CloseButton, emptyRendering.CloseButton)
	assert.Equal(t, omittedRendering.CloseButton, omittedSection.CloseButton)

	require.NotNil(t, omittedSection.Container)
	require.NotNil(t, omittedSection.Container.ImpressionTracking)
	assert.Equal(t, rendering.ContainerFormatInterstitial, omittedSection.Container.Format)
}

// A bad asset URL drops only that entry; the rest of the endcards customization (and the
// other, valid assets) survives.
func TestParseFromBidExt_endcardBadAssetDropsOnlyThatEntry(t *testing.T) {
	t.Parallel()

	bidExt := []byte(`{
		"rendering": {
			"endcards": {
				"cta_text": "Play Now",
				"assets": [
					{ "url": "https://cdn.example.com/good.png" },
					{ "url": "not-a-url" }
				]
			}
		}
	}`)

	got := rendering.ParseFromBidExt(bidExt, testDemandID)
	require.NotNil(t, got)
	require.NotNil(t, got.EndCards)
	assert.Equal(t, "Play Now", got.EndCards.CTAText)
	require.Len(t, got.EndCards.Assets, 1)
	assert.Equal(t, "https://cdn.example.com/good.png", got.EndCards.Assets[0].URL)
}

// A validation failure is logged with the offending section and demand ID so misconfigured
// bid responses are discoverable rather than silently swallowed.
func TestParseFromBidExt_logsSectionFailure(t *testing.T) {
	var buf strings.Builder
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(prev)

	bidExt := []byte(`{"rendering": {"creative": {"type": "unsupported"}}}`)
	rendering.ParseFromBidExt(bidExt, testDemandID)

	logged := buf.String()
	assert.Contains(t, logged, "test-dsp")
	assert.Contains(t, logged, "Creative")
}

// Backfilling an omitted section with defaults is normal, expected behavior, not a
// misconfiguration - it must not log anything. Only genuine validation failures should.
func TestParseFromBidExt_doesNotLogOnPlainOmission(t *testing.T) {
	var buf strings.Builder
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(prev)

	// Only close_button is set; endcards, container, creative, and store_kit are all
	// silently backfilled with defaults.
	bidExt := []byte(`{"rendering": {"close_button": {"style": "icon_circle"}}}`)
	rendering.ParseFromBidExt(bidExt, testDemandID)

	assert.Empty(t, buf.String())
}

func TestParseFromBidExt_noRendering(t *testing.T) {
	t.Parallel()

	bidExt := []byte(`{"signaldata": "abc"}`)

	got := rendering.ParseFromBidExt(bidExt, testDemandID)
	require.NotNil(t, got)
	require.NotNil(t, got.CloseButton)
	assert.True(t, *got.CloseButton.Visible)
	assert.Equal(t, rendering.CloseButtonStyleIconX, got.CloseButton.Style)

	defaults := rendering.ParseFromBidExt(nil, testDemandID)
	require.NotNil(t, defaults)
	require.NotNil(t, defaults.Container)
	assert.Equal(t, rendering.ContainerFormatInterstitial, defaults.Container.Format)
	require.NotNil(t, defaults.Creative)
	assert.Equal(t, rendering.CreativeTypeStaticImage, defaults.Creative.Type)
	assert.Equal(t, rendering.CreativeSourceADM, defaults.Creative.Source)
	assert.Equal(t, rendering.CreateMRaidVersionV3, defaults.Creative.MRAIDVersion)
	assert.Equal(t, rendering.CreativeVASTVersionV42, defaults.Creative.VASTVersion)
	assert.Equal(t, rendering.CreativePreloadStrategyEager, defaults.Creative.PreloadStrategy)
}

func TestDefaultConfig_validates(t *testing.T) {
	t.Parallel()

	err := rendering.Validate(rendering.DefaultConfig())
	assert.NoError(t, err)
}

func TestDefaultConfig_usesTypedEnumValues(t *testing.T) {
	t.Parallel()

	cfg := rendering.DefaultConfig()
	require.NotNil(t, cfg.EndCards)
	assert.Equal(t, rendering.EndCardsLayoutSingle, cfg.EndCards.Layout)
	require.NotNil(t, cfg.StoreKit)
	assert.Equal(t, rendering.StoreKitTriggerOnEndCard, cfg.StoreKit.Trigger)
	assert.Equal(t, rendering.StoreKitPositionBottom, cfg.StoreKit.Position)
}

func TestParseFromBidExt_serializesForSDK(t *testing.T) {
	t.Parallel()

	bidExt := []byte(`{
		"rendering": {
			"creative": { "type": "vast", "vast_version": "4.2" }
		}
	}`)

	got := rendering.ParseFromBidExt(bidExt, testDemandID)
	require.NotNil(t, got)

	out, err := json.Marshal(got)
	require.NoError(t, err)
	assert.Contains(t, string(out), `"type":"vast"`)
	assert.Contains(t, string(out), `"vast_version":"4.2"`)
}

func TestValidate_nilConfig(t *testing.T) {
	t.Parallel()

	assert.NoError(t, rendering.Validate(nil))
}

func TestApplyDefaults_nil(t *testing.T) {
	t.Parallel()

	assert.NoError(t, rendering.ApplyDefaults(nil))
}

func TestValidate_closeButtonHexColor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		color string
	}{
		{name: "invalid length", color: "#FFFF"},
		{name: "missing hash", color: "FFFFFF"},
		{name: "invalid characters", color: "#GGGGGG"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := rendering.Config{
				CloseButton: &rendering.CloseButtonConfig{
					Style: rendering.CloseButtonStyleIconX,
					Color: tt.color,
				},
				Creative: &rendering.CreativeConfig{Type: rendering.CreativeTypeVAST},
			}
			err := rendering.Validate(&cfg)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "CloseButton.Color")
			assert.Contains(t, err.Error(), "hexcolor")
		})
	}
}

func TestValidate_closeButtonHexColor_allowsShortForm(t *testing.T) {
	t.Parallel()

	cfg := rendering.Config{
		CloseButton: &rendering.CloseButtonConfig{
			Style: rendering.CloseButtonStyleIconX,
			Color: "#FFF",
		},
		Creative: &rendering.CreativeConfig{Type: rendering.CreativeTypeVAST},
	}
	assert.NoError(t, rendering.Validate(&cfg))
}

func TestValidate_customStyleWithValidAssetURL(t *testing.T) {
	t.Parallel()

	cfg := rendering.Config{
		CloseButton: &rendering.CloseButtonConfig{
			Style:          rendering.CloseButtonStyleCustom,
			CustomAssetURL: "https://cdn.example.com/icon.png",
		},
		Creative: &rendering.CreativeConfig{Type: rendering.CreativeTypeVAST},
	}
	assert.NoError(t, rendering.Validate(&cfg))
}

func TestValidate_customAssetURLInvalidWhenNotCustom(t *testing.T) {
	t.Parallel()

	cfg := rendering.Config{
		CloseButton: &rendering.CloseButtonConfig{
			Style:          rendering.CloseButtonStyleIconX,
			CustomAssetURL: "not-a-url",
		},
		Creative: &rendering.CreativeConfig{Type: rendering.CreativeTypeVAST},
	}
	err := rendering.Validate(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CloseButton.CustomAssetURL")
	assert.Contains(t, err.Error(), "custom_asset_url")
}

func TestValidate_endcardAssetRequiresHTTPURL(t *testing.T) {
	t.Parallel()

	cfg := rendering.Config{
		EndCards: &rendering.EndCardsConfig{
			Assets: []rendering.EndcardAsset{{URL: "not-a-url"}},
		},
		Creative: &rendering.CreativeConfig{Type: rendering.CreativeTypeVAST},
	}
	err := rendering.Validate(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "EndCards.Assets")
	assert.Contains(t, err.Error(), "httpurl")
}

func TestParseFromBidExt_invalidJSON(t *testing.T) {
	t.Parallel()

	got := rendering.ParseFromBidExt([]byte(`{invalid`), testDemandID)
	require.NotNil(t, got)
	require.NotNil(t, got.CloseButton)
	assert.Equal(t, rendering.CloseButtonStyleIconX, got.CloseButton.Style)
}

func TestParseFromBidExt_nullRendering(t *testing.T) {
	t.Parallel()

	got := rendering.ParseFromBidExt([]byte(`{"rendering": null}`), testDemandID)
	require.NotNil(t, got)
	require.NotNil(t, got.CloseButton)
	assert.Equal(t, rendering.CloseButtonStyleIconX, got.CloseButton.Style)
}

func TestParseFromBidExt_invalidCloseButtonResetsSection(t *testing.T) {
	t.Parallel()

	bidExt := []byte(`{
		"rendering": {
			"close_button": { "style": "icon_circle", "opacity": 2 },
			"creative": { "type": "vast" }
		}
	}`)

	got := rendering.ParseFromBidExt(bidExt, testDemandID)
	require.NotNil(t, got.CloseButton)
	assert.Equal(t, rendering.CloseButtonStyleIconX, got.CloseButton.Style)
	require.NotNil(t, got.CloseButton.Opacity)
	assert.Equal(t, 0.9, *got.CloseButton.Opacity)
	require.NotNil(t, got.Creative)
	assert.Equal(t, rendering.CreativeTypeVAST, got.Creative.Type)
}

func TestParseFromBidExt_invalidContainerResetsSection(t *testing.T) {
	t.Parallel()

	bidExt := []byte(`{
		"rendering": {
			"container": { "format": "not_a_format", "background_color": "#111111" },
			"creative": { "type": "vast" }
		}
	}`)

	got := rendering.ParseFromBidExt(bidExt, testDemandID)
	require.NotNil(t, got.Container)
	assert.Equal(t, rendering.ContainerFormatInterstitial, got.Container.Format)
	assert.Equal(t, "#000000", got.Container.BackgroundColor)
	require.NotNil(t, got.Creative)
	assert.Equal(t, rendering.CreativeTypeVAST, got.Creative.Type)
}

func TestParseFromBidExt_invalidEndCardsSectionResetsWholeSection(t *testing.T) {
	t.Parallel()

	bidExt := []byte(`{
		"rendering": {
			"endcards": { "layout": "not_a_layout", "cta_text": "Go" },
			"creative": { "type": "vast" }
		}
	}`)

	got := rendering.ParseFromBidExt(bidExt, testDemandID)
	require.NotNil(t, got.EndCards)
	assert.Equal(t, rendering.EndCardsLayoutSingle, got.EndCards.Layout)
	assert.Equal(t, "Install Now", got.EndCards.CTAText)
	require.NotNil(t, got.Creative)
	assert.Equal(t, rendering.CreativeTypeVAST, got.Creative.Type)
}

func intPtr(v int) *int {
	return &v
}
