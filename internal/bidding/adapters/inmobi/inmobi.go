package inmobi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v19/adcom1"
	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

type InMobiAdapter struct {
	AppID       string
	PlacementID string
}

var _ adapters.BidderInterface = (*InMobiAdapter)(nil)

func (a *InMobiAdapter) banner(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	imp, _ := adapters.BuildBannerImp(auctionRequest, adapters.BannerImpOptions{
		API: []adcom1.APIFramework{3, 5}, // MRAID 1.0, MRAID 2.0
	})
	return imp
}

func (a *InMobiAdapter) interstitial(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	return adapters.BuildInterstitialImp(auctionRequest, adapters.InterstitialImpOptions{
		BannerAPI: []adcom1.APIFramework{3, 5}, // MRAID 1.0, MRAID 2.0
	})
}

// rewarded stays custom: rich VAST constraints + AboveFold Pos + is_rewarded Ext.
func (a *InMobiAdapter) rewarded(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	w, h := adapters.ResolveFullscreenSize(
		string(auctionRequest.Device.Type),
		auctionRequest.AdObject.IsPortrait(),
		true,
	)
	return &openrtb2.Imp{
		Instl: 0,
		Video: &openrtb2.Video{
			W:           w,
			H:           h,
			MIMEs:       []string{"video/mp4"},
			MinDuration: 0,
			MaxDuration: 6000,
			Protocols:   []adcom1.MediaCreativeSubtype{2, 3, 5, 6}, // VAST 2.0, VAST 3.0, VAST 4.0 Wrapper, VAST 4.1 Wrapper
			StartDelay:  adcom1.StartDelay(0).Ptr(),                // Pre-roll
			API:         []adcom1.APIFramework{1, 2, 3, 5, 6, 7},   // VPAID 1.0, VPAID 2.0, MRAID 1.0, MRAID 2.0, MRAID 3.0, OMID 1.0
			Pos:         adcom1.PositionAboveFold.Ptr(),
		},
		Ext: json.RawMessage(`{"is_rewarded": true}`),
	}
}

func (a *InMobiAdapter) BuildImpression(_ openrtb.BidRequest, auctionRequest *schema.AuctionRequest) (*openrtb2.Imp, adapters.RTBRequestOptions, error) {
	if a.PlacementID == "" {
		return nil, adapters.RTBRequestOptions{}, errors.New("PlacementID is empty")
	}

	var imp *openrtb2.Imp
	switch auctionRequest.AdObject.Type() {
	case ad.BannerType:
		imp = a.banner(auctionRequest)
	case ad.InterstitialType:
		imp = a.interstitial(auctionRequest)
	case ad.RewardedType:
		imp = a.rewarded(auctionRequest)
	default:
		return nil, adapters.RTBRequestOptions{}, errors.New("unknown impression type")
	}

	opts := adapters.RTBRequestOptions{
		TagID: a.PlacementID,
		AppID: a.AppID,
	}
	if token, exists := auctionRequest.AdObject.Demands[adapter.InmobiKey]["token"]; exists {
		if tokenStr, ok := token.(string); ok {
			opts.BuyerUID = tokenStr
		}
	}

	return imp, opts, nil
}

func (a *InMobiAdapter) ExecuteRequest(ctx context.Context, client *http.Client, request openrtb.BidRequest) *adapters.DemandResponse {
	return adapters.ExecuteRTBRequest(ctx, client, request, adapters.ExecuteRTBOptions{
		DemandID:    adapter.InmobiKey,
		URL:         "https://api.w.inmobi.com/ortb/imsdk",
		PlacementID: a.PlacementID,
		Headers:     http.Header{"X-OpenRTB-Version": {"2.5"}},
	})
}

// Builder builds a new instance of the InMobi adapter for the given bidder with the given config.
func Builder(cfg adapter.ProcessedConfigsMap, client *http.Client) (*adapters.Bidder, error) {
	inmobiCfg := cfg[adapter.InmobiKey]

	appID, ok := inmobiCfg["app_id"].(string)
	if !ok || appID == "" {
		return nil, fmt.Errorf("missing app_id param for %s adapter", adapter.InmobiKey)
	}

	placementID, ok := inmobiCfg["placement_id"].(string)
	if !ok || placementID == "" {
		return nil, fmt.Errorf("missing placement_id param for %s adapter", adapter.InmobiKey)
	}

	adpt := &InMobiAdapter{
		AppID:       appID,
		PlacementID: placementID,
	}

	bidder := adapters.Bidder{
		Adapter: adpt,
		Client:  client,
	}

	return &bidder, nil
}
