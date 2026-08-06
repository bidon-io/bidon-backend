package vkads

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

type VKAdsAdapter struct {
	TagID string
	AppID string
}

var _ adapters.BidderInterface = (*VKAdsAdapter)(nil)

const (
	rewardedWidth  int64 = 1920
	rewardedHeight int64 = 1080
)

func (a *VKAdsAdapter) banner(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	imp, _ := adapters.BuildBannerImp(auctionRequest, adapters.BannerImpOptions{})
	return imp
}

// interstitial stays custom: Pos-only fullscreen Banner without W/H.
func (a *VKAdsAdapter) interstitial() *openrtb2.Imp {
	return &openrtb2.Imp{
		Instl: 1,
		Banner: &openrtb2.Banner{
			Pos: adcom1.PositionFullScreen.Ptr(),
		},
	}
}

// rewarded stays custom: fixed 1920×1080 Banner (not fullscreen device sizes).
func (a *VKAdsAdapter) rewarded() *openrtb2.Imp {
	w, h := rewardedWidth, rewardedHeight
	return &openrtb2.Imp{
		Banner: &openrtb2.Banner{
			W: &w,
			H: &h,
		},
	}
}

func (a *VKAdsAdapter) BuildImpression(_ openrtb.BidRequest, auctionRequest *schema.AuctionRequest) (*openrtb2.Imp, adapters.RTBRequestOptions, error) {
	if a.AppID == "" {
		return nil, adapters.RTBRequestOptions{}, errors.New("AppID is empty")
	}
	if a.TagID == "" {
		return nil, adapters.RTBRequestOptions{}, errors.New("TagID is empty")
	}
	token, ok := auctionRequest.AdObject.Demands[adapter.VKAdsKey]["token"].(string)
	if !ok || token == "" {
		return nil, adapters.RTBRequestOptions{}, errors.New("token is empty")
	}

	var imp *openrtb2.Imp
	switch auctionRequest.AdObject.Type() {
	case ad.BannerType:
		imp = a.banner(auctionRequest)
	case ad.InterstitialType:
		imp = a.interstitial()
	case ad.RewardedType:
		imp = a.rewarded()
	default:
		return nil, adapters.RTBRequestOptions{}, errors.New("unknown impression type")
	}

	// VK Ads historically omits DisplayManager and Secure; keep those defaults off.
	return imp, adapters.RTBRequestOptions{
		TagID:              a.TagID,
		AppID:              a.AppID,
		OmitSecure:         true,
		OmitDisplayManager: true,
	}, nil
}

func (a *VKAdsAdapter) EnrichOpenRTBRequest(request *openrtb.BidRequest, auctionRequest *schema.AuctionRequest) error {
	token, _ := auctionRequest.AdObject.Demands[adapter.VKAdsKey]["token"].(string)
	request.User = &openrtb.User{
		ID:  auctionRequest.User.IDG,
		Ext: json.RawMessage(fmt.Sprintf(`{"buyeruid": "%s"}`, token)),
	}
	request.Ext = json.RawMessage(`{"pid":111}`)
	return nil
}

func (a *VKAdsAdapter) ExecuteRequest(ctx context.Context, client *http.Client, request openrtb.BidRequest) *adapters.DemandResponse {
	return adapters.ExecuteRTBRequest(ctx, client, request, adapters.ExecuteRTBOptions{
		DemandID: adapter.VKAdsKey,
		URL:      "https://ad.mail.ru/api/bid",
		TagID:    a.TagID,
	})
}

// Builder builds a new instance of the VKAds adapter for the given bidder with the given config.
func Builder(cfg adapter.ProcessedConfigsMap, client *http.Client) (*adapters.Bidder, error) {
	vkCfg := cfg[adapter.VKAdsKey]

	appID, ok := vkCfg["app_id"].(string)
	if !ok || appID == "" {
		appID = ""
	}

	tagID, ok := vkCfg["tag_id"].(string)
	if !ok {
		tagID = ""
	}

	adpt := &VKAdsAdapter{
		AppID: appID,
		TagID: tagID,
	}

	bidder := &adapters.Bidder{
		Adapter: adpt,
		Client:  client,
	}

	return bidder, nil
}
