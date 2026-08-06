package vungle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

type VungleAdapter struct {
	SellerID string
	AppID    string
	TagID    string
}

var _ adapters.BidderInterface = (*VungleAdapter)(nil)

func (a *VungleAdapter) banner(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	imp, _ := adapters.BuildBannerImp(auctionRequest, adapters.BannerImpOptions{})
	return imp
}

func (a *VungleAdapter) interstitial(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	return adapters.BuildInterstitialImp(auctionRequest, adapters.InterstitialImpOptions{})
}

// rewarded stays custom: Video.Ext rewarded flag without Instl/Pos defaults.
func (a *VungleAdapter) rewarded(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	w, h := adapters.ResolveFullscreenSize(
		string(auctionRequest.Device.Type),
		auctionRequest.AdObject.IsPortrait(),
		true,
	)
	return &openrtb2.Imp{
		Video: &openrtb2.Video{
			W:     w,
			H:     h,
			MIMEs: []string{"video/mp4"},
			Ext:   json.RawMessage(`{"rewarded": 1}`),
		},
	}
}

func (a *VungleAdapter) BuildImpression(_ openrtb.BidRequest, auctionRequest *schema.AuctionRequest) (*openrtb2.Imp, adapters.RTBRequestOptions, error) {
	if a.TagID == "" {
		return nil, adapters.RTBRequestOptions{}, errors.New("TagID is empty")
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

	vungleData := make(map[string]interface{})
	vungleData["bid_token"] = auctionRequest.AdObject.Demands[adapter.VungleKey]["token"]

	extStructure := &map[string]interface{}{}
	_ = json.Unmarshal(imp.Ext, extStructure)
	(*extStructure)["vungle"] = vungleData
	raw, _ := json.Marshal(extStructure)
	imp.Ext = raw

	return imp, adapters.RTBRequestOptions{
		TagID:       a.TagID,
		AppID:       a.AppID,
		PublisherID: a.SellerID,
	}, nil
}

func (a *VungleAdapter) ExecuteRequest(ctx context.Context, client *http.Client, request openrtb.BidRequest) *adapters.DemandResponse {
	return adapters.ExecuteRTBRequest(ctx, client, request, adapters.ExecuteRTBOptions{
		DemandID: adapter.VungleKey,
		URL:      "https://rtb.ads.vungle.com/bid/t/8ea3e9a",
		TagID:    a.TagID,
		Headers:  http.Header{"X-OpenRTB-Version": {"2.5"}},
	})
}

// Builder builds a new instance of the Vungle adapter for the given bidder with the given config.
func Builder(cfg adapter.ProcessedConfigsMap, client *http.Client) (*adapters.Bidder, error) {
	vCfg := cfg[adapter.VungleKey]

	sellerID, ok := vCfg["seller_id"].(string)
	if !ok || sellerID == "" {
		return nil, fmt.Errorf("missing seller_id param for %s adapter", adapter.VungleKey)
	}
	appID, ok := vCfg["app_id"].(string)
	if !ok || appID == "" {
		return nil, fmt.Errorf("missing app_id param for %s adapter", adapter.VungleKey)
	}
	tagID, ok := vCfg["tag_id"].(string)
	if !ok {
		tagID = ""
	}

	adpt := &VungleAdapter{
		SellerID: sellerID,
		AppID:    appID,
		TagID:    tagID,
	}

	bidder := adapters.Bidder{
		Adapter: adpt,
		Client:  client,
	}

	return &bidder, nil
}
