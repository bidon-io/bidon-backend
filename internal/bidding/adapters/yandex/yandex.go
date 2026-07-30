package yandex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

const yandexEndpoint = "https://mobile.yandexadexchange.net/openbidding?ssp-id=99048272"

type YandexAdapter struct {
	AdUnitID string
}

var bannerFormats = map[ad.Format][2]int64{
	ad.BannerFormat:      {320, 50},
	ad.LeaderboardFormat: {728, 90},
	ad.MRECFormat:        {300, 250},
	ad.AdaptiveFormat:    {320, 50},
	ad.EmptyFormat:       {320, 50}, // Default
}

func (a *YandexAdapter) banner(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
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

func (a *YandexAdapter) interstitial() *openrtb2.Imp {
	return &openrtb2.Imp{
		Instl: 1,
	}
}

func (a *YandexAdapter) rewarded() *openrtb2.Imp {
	return &openrtb2.Imp{
		Instl: 0,
		Video: &openrtb2.Video{
			MIMEs: []string{"video/mp4"},
		},
	}
}

func (a *YandexAdapter) CreateRequest(request openrtb.BidRequest, auctionRequest *schema.AuctionRequest) (openrtb.BidRequest, error) {
	if a.AdUnitID == "" {
		return request, errors.New("AdUnitID is empty")
	}

	demandData, ok := auctionRequest.AdObject.Demands[adapter.YandexKey]
	if !ok {
		return request, errors.New("yandex demand data is missing")
	}

	token, ok := demandData["token"].(string)
	if !ok || token == "" {
		return request, errors.New("yandex bidder token is empty")
	}

	var imp *openrtb2.Imp
	var adTypeString string
	var rwdd int8

	switch auctionRequest.AdObject.Type() {
	case ad.BannerType:
		imp = a.banner(auctionRequest)
		adTypeString = "banner"
		rwdd = 0
	case ad.InterstitialType:
		imp = a.interstitial()
		adTypeString = "interstitial"
		rwdd = 0
	case ad.RewardedType:
		imp = a.rewarded()
		adTypeString = "rewarded"
		rwdd = 1
	default:
		return request, errors.New("unknown impression type")
	}

	imp.Rwdd = rwdd
	impExt, err := json.Marshal(map[string]any{
		"ad_type": adTypeString,
	})
	if err != nil {
		return request, err
	}
	imp.Ext = impExt

	request = adapters.BuildRTBRequest(request, auctionRequest, adapter.YandexKey, imp, adapters.RTBRequestOptions{
		TagID: a.AdUnitID,
	})

	request.User = &openrtb.User{
		Data: []openrtb.Data{
			{
				Segment: []openrtb.Segment{
					{
						Signal: token,
					},
				},
			},
		},
	}

	return request, nil
}

func (a *YandexAdapter) ExecuteRequest(ctx context.Context, client *http.Client, request openrtb.BidRequest) *adapters.DemandResponse {
	dr := &adapters.DemandResponse{
		DemandID:  adapter.YandexKey,
		RequestID: request.ID,
		TagID:     a.AdUnitID,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		dr.Error = err
		return dr
	}
	dr.RawRequest = string(requestBody)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, yandexEndpoint, bytes.NewBuffer(requestBody))
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

	return dr
}

// Builder builds a new instance of the Yandex adapter for the given bidder with the given config.
func Builder(cfg adapter.ProcessedConfigsMap, client *http.Client) (*adapters.Bidder, error) {
	yandexCfg := cfg[adapter.YandexKey]

	adUnitID, ok := yandexCfg["ad_unit_id"].(string)
	if !ok {
		adUnitID = ""
	}

	adpt := &YandexAdapter{
		AdUnitID: adUnitID,
	}

	bidder := &adapters.Bidder{
		Adapter: adpt,
		Client:  client,
	}

	return bidder, nil
}
