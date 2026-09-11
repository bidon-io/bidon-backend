package dspsim

import (
	"errors"
	"strings"

	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/ad"
)

// NoBidReason explains why the simulator answered 204. Every reason is logged
// and surfaced on /debug.
type NoBidReason string

const (
	ReasonNoImpression        NoBidReason = "no_impression"
	ReasonUnknownBundle       NoBidReason = "unknown_bundle"
	ReasonNoConfigForAdType   NoBidReason = "no_config_for_adtype"
	ReasonFormatNotConfigured NoBidReason = "format_not_configured"
	ReasonFloorTooHigh        NoBidReason = "floor_too_high"
	ReasonDemandNotConfigured NoBidReason = "demand_not_configured"
	ReasonNoCreative          NoBidReason = "no_creative"
	ReasonRandomNoBid         NoBidReason = "random_no_bid"
)

// ErrNoImpression is returned when a bid request carries no impression at all.
var ErrNoImpression = errors.New("bid request has no impression")

// RequestSummary is everything the simulator infers from an inbound bid
// request. Ad type and format are derived from the shape of imp[0], because
// OpenRTB carries no explicit ad type: this mirrors what the bidon adapters
// build (see internal/bidding/adapters/adikteev/adikteev.go).
type RequestSummary struct {
	RequestID  string    `json:"request_id"`
	Bundle     string    `json:"bundle"`
	DSP        string    `json:"dsp"`
	ImpID      string    `json:"imp_id"`
	AdType     ad.Type   `json:"ad_type"`
	Format     ad.Format `json:"format"`
	Width      int64     `json:"w"`
	Height     int64     `json:"h"`
	Floor      float64   `json:"floor"`
	Currency   string    `json:"floor_cur"`
	HasVideo   bool      `json:"has_video"`
	HasBanner  bool      `json:"has_banner"`
	Fullscreen bool      `json:"fullscreen"`
	// AdTypeAmbiguous is set for fullscreen impressions, where interstitial and
	// rewarded are indistinguishable on the wire.
	AdTypeAmbiguous bool `json:"ad_type_ambiguous"`
}

// Match is a successful pairing of a bid request with a configured auction.
type Match struct {
	Summary          RequestSummary
	Auction          *ConfiguredAuction
	DemandConfigured bool
}

// Describe extracts the simulator's view of a bid request.
func Describe(req *openrtb2.BidRequest) (RequestSummary, error) {
	if req == nil || len(req.Imp) == 0 {
		return RequestSummary{}, ErrNoImpression
	}

	imp := req.Imp[0]
	summary := RequestSummary{
		RequestID: req.ID,
		ImpID:     imp.ID,
		DSP:       strings.ToLower(strings.TrimSpace(imp.DisplayManager)),
		Floor:     imp.BidFloor,
		Currency:  imp.BidFloorCur,
		HasVideo:  imp.Video != nil,
		HasBanner: imp.Banner != nil,
		Bundle:    appBundle(req.App),
	}

	summary.Width, summary.Height = impSize(imp)
	summary.Fullscreen = imp.Instl == 1 || isFullscreenSize(summary.Width, summary.Height)

	switch {
	case summary.Fullscreen:
		// Interstitial and rewarded produce the same imp; the catalog decides
		// which one this app has configured.
		summary.AdType = ad.InterstitialType
		summary.AdTypeAmbiguous = true
		summary.Format = ad.EmptyFormat
	default:
		summary.AdType = ad.BannerType
		summary.Format = bannerFormat(summary.Width, summary.Height)
	}

	return summary, nil
}

func appBundle(app *openrtb2.App) string {
	if app == nil {
		return ""
	}
	for _, candidate := range []string{app.Bundle, app.ID} {
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

func impSize(imp openrtb2.Imp) (int64, int64) {
	if imp.Banner != nil {
		if imp.Banner.W != nil && imp.Banner.H != nil {
			return *imp.Banner.W, *imp.Banner.H
		}
		if len(imp.Banner.Format) > 0 {
			return imp.Banner.Format[0].W, imp.Banner.Format[0].H
		}
	}
	if imp.Video != nil {
		return imp.Video.W, imp.Video.H
	}
	return 0, 0
}

// isFullscreenSize treats anything taller than an MREC as fullscreen, so that a
// video-only or oversized imp sent without instl is still recognised.
func isFullscreenSize(_, h int64) bool {
	return h > 250
}

func bannerFormat(w, h int64) ad.Format {
	switch {
	case w == 300 && h == 250:
		return ad.MRECFormat
	case w == 728 && h == 90:
		return ad.LeaderboardFormat
	case w == 320 && h == 50:
		return ad.BannerFormat
	default:
		return ad.BannerFormat
	}
}

// Matcher resolves a request summary against the catalog.
type Matcher struct {
	Catalog  *CatalogStore
	MaxPrice float64
	// StrictDemand turns an unconfigured demand from a warning into a no-bid.
	StrictDemand bool
}

// Match pairs a summary with a configured auction. adTypeOverride, when set,
// pins the ad type instead of letting the catalog resolve the fullscreen
// ambiguity; it exists so manual testing is deterministic.
func (m *Matcher) Match(summary RequestSummary, adTypeOverride ad.Type) (*Match, NoBidReason) {
	catalog := m.Catalog.Get()

	if summary.Bundle == "" || !catalog.KnowsBundle(summary.Bundle) {
		return nil, ReasonUnknownBundle
	}

	candidates := m.adTypeCandidates(summary, adTypeOverride)

	var formatMismatch bool
	for _, adType := range candidates {
		auctions := catalog.Lookup(summary.Bundle, adType)
		if len(auctions) == 0 {
			continue
		}
		for _, auction := range auctions {
			if !auction.SupportsFormat(summary.Format) {
				formatMismatch = true
				continue
			}

			demandConfigured := summary.DSP == "" || auction.HasDemand(summary.DSP)
			if !demandConfigured && m.StrictDemand {
				return nil, ReasonDemandNotConfigured
			}

			if m.MaxPrice > 0 && summary.Floor > m.MaxPrice {
				return nil, ReasonFloorTooHigh
			}

			resolved := summary
			resolved.AdType = adType
			return &Match{
				Summary:          resolved,
				Auction:          auction,
				DemandConfigured: demandConfigured,
			}, ""
		}
	}

	if formatMismatch {
		return nil, ReasonFormatNotConfigured
	}
	return nil, ReasonNoConfigForAdType
}

// adTypeCandidates orders the ad types to try. Fullscreen impressions may be
// either interstitial or rewarded, so both are tried in turn.
func (m *Matcher) adTypeCandidates(summary RequestSummary, override ad.Type) []ad.Type {
	if override != ad.UnknownType {
		return []ad.Type{override}
	}
	if !summary.AdTypeAmbiguous {
		return []ad.Type{summary.AdType}
	}
	// A video-capable imp is more likely rewarded in practice, but the ordering
	// only decides ties when an app configures both.
	if summary.HasVideo && !summary.HasBanner {
		return []ad.Type{ad.RewardedType, ad.InterstitialType}
	}
	return []ad.Type{ad.InterstitialType, ad.RewardedType}
}

// ParseAdType maps a query-string override onto a domain ad type.
func ParseAdType(s string) ad.Type {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "banner":
		return ad.BannerType
	case "interstitial":
		return ad.InterstitialType
	case "rewarded":
		return ad.RewardedType
	default:
		return ad.UnknownType
	}
}
