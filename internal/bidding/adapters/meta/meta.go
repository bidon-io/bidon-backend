package meta

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/prebid/openrtb/v19/adcom1"
	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

type MetaAdapter struct {
	AppID      string
	AppSecret  string
	PlatformID string
	TagID      string
}

var _ adapters.BidderInterface = (*MetaAdapter)(nil)

var bannerFormats = map[ad.Format][2]int64{
	ad.BannerFormat:      {320, 50},
	ad.LeaderboardFormat: {728, 90},
	ad.MRECFormat:        {300, 250},
	ad.AdaptiveFormat:    {0, 50},
	ad.EmptyFormat:       {320, 50}, // Default
}

func (a *MetaAdapter) banner(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	size := bannerFormats[auctionRequest.AdObject.Format()]

	if auctionRequest.AdObject.IsAdaptive() && auctionRequest.Device.IsTablet() {
		size = bannerFormats[ad.LeaderboardFormat]
	}

	w, h := size[0], size[1]

	return &openrtb2.Imp{
		Instl: 0,
		Banner: &openrtb2.Banner{
			W:   &w,
			H:   &h,
			Pos: adcom1.PositionAboveFold.Ptr(),
		},
	}
}

func (a *MetaAdapter) interstitial(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	size := adapters.FullscreenFormats[string(auctionRequest.Device.Type)]
	w, h := size[0], size[1]
	if !auctionRequest.AdObject.IsPortrait() {
		w, h = h, w
	}
	return &openrtb2.Imp{
		Instl: 1,
		Banner: &openrtb2.Banner{
			W:   &w,
			H:   &h,
			Pos: adcom1.PositionFullScreen.Ptr(),
		},
	}
}

func (a *MetaAdapter) rewarded(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	size := adapters.FullscreenFormats[string(auctionRequest.Device.Type)]
	w, h := size[0], size[1]
	if !auctionRequest.AdObject.IsPortrait() {
		w, h = h, w
	}
	return &openrtb2.Imp{
		Video: &openrtb2.Video{
			W:   w,
			H:   h,
			Ext: json.RawMessage(`{"videotype": "rewarded"}`),
		},
	}
}

func (a *MetaAdapter) timeoutURL(platformID string) string {
	return "https://www.facebook.com/audiencenetwork/nurl/?partner=" + platformID + "&app=" + a.AppID + "&auction=${AUCTION_ID}&ortb_loss_code=2"
}

func (a *MetaAdapter) BuildImpression(_ openrtb.BidRequest, auctionRequest *schema.AuctionRequest) (*openrtb2.Imp, adapters.RTBRequestOptions, error) {
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

	opts := adapters.RTBRequestOptions{
		TagID:       a.TagID,
		PublisherID: a.AppID,
	}
	if token, ok := auctionRequest.AdObject.Demands[adapter.MetaKey]["token"].(string); ok {
		opts.BuyerUID = token
	}

	return imp, opts, nil
}

func (a *MetaAdapter) EnrichOpenRTBRequest(request *openrtb.BidRequest, _ *schema.AuctionRequest) error {
	ext, err := json.Marshal(map[string]any{
		"platformid":        a.PlatformID,
		"authentication_id": calculateHMACSHA256(request.ID, a.AppSecret),
	})
	if err != nil {
		return err
	}
	request.Ext = ext
	return nil
}

func (a *MetaAdapter) ExecuteRequest(ctx context.Context, client *http.Client, request openrtb.BidRequest) *adapters.DemandResponse {
	dr := &adapters.DemandResponse{
		DemandID:   adapter.MetaKey,
		RequestID:  request.ID,
		TimeoutURL: a.timeoutURL(a.PlatformID),
		TagID:      a.TagID,
	}
	requestBody, err := json.Marshal(request)
	if err != nil {
		dr.Error = err
		return dr
	}
	dr.RawRequest = string(requestBody)

	url := "https://an.facebook.com/" + a.PlatformID + "/placementbid.ortb"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(requestBody))
	if err != nil {
		dr.Error = err
		return dr
	}
	httpReq.Header.Add("Content-Type", "application/json")

	httpResp, err := client.Do(httpReq)
	if err != nil {
		dr.Error = err
		return dr
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		dr.Error = err
		return dr
	}

	dr.RawResponse = string(respBody)
	dr.Status = httpResp.StatusCode
	if dr.Status == http.StatusBadRequest {
		dr.Error = errors.New(httpResp.Header.Get("X-Fb-An-Errors"))
	}

	return dr
}

// Builder builds a new instance of the Meta adapter for the given bidder with the given config.
func Builder(cfg adapter.ProcessedConfigsMap, client *http.Client) (*adapters.Bidder, error) {
	mCfg := cfg[adapter.MetaKey]

	appID, ok := mCfg["app_id"].(string)
	if !ok || appID == "" {
		return nil, fmt.Errorf("missing app_id param for %s adapter", adapter.MetaKey)
	}
	appSecret, ok := mCfg["app_secret"].(string)
	if !ok || appID == "" {
		return nil, fmt.Errorf("missing app_secret param for %s adapter", adapter.MetaKey)
	}
	platformID, ok := mCfg["platform_id"].(string)
	if !ok || platformID == "" {
		return nil, fmt.Errorf("missing platform_id param for %s adapter", adapter.MetaKey)
	}
	tagID, ok := mCfg["tag_id"].(string)
	if !ok {
		tagID = ""
	}

	adpt := &MetaAdapter{
		AppID:      appID,
		AppSecret:  appSecret,
		PlatformID: platformID,
		TagID:      tagID,
	}

	bidder := adapters.Bidder{
		Adapter: adpt,
		Client:  client,
	}

	return &bidder, nil
}

func calculateHMACSHA256(data, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))

	return hex.EncodeToString(h.Sum(nil))
}
