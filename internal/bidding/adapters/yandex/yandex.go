package yandex

import (
	"encoding/json"
	"errors"
	"net/http"

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

var _ adapters.BidderInterface = (*YandexAdapter)(nil)

func (a *YandexAdapter) banner(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	imp, _ := adapters.BuildBannerImp(auctionRequest, adapters.BannerImpOptions{})
	return imp
}

// interstitial stays custom: Instl-only Imp without Banner/Video.
func (a *YandexAdapter) interstitial() *openrtb2.Imp {
	return &openrtb2.Imp{
		Instl: 1,
	}
}

// rewarded stays custom: minimal mp4 Video; Rwdd/Ext set in CreateRequest.
func (a *YandexAdapter) rewarded() *openrtb2.Imp {
	return &openrtb2.Imp{
		Instl: 0,
		Video: &openrtb2.Video{
			MIMEs: []string{"video/mp4"},
		},
	}
}

func (a *YandexAdapter) BuildImpression(_ openrtb.BidRequest, auctionRequest *schema.AuctionRequest) (*openrtb2.Imp, adapters.RTBRequestOptions, error) {
	if a.AdUnitID == "" {
		return nil, adapters.RTBRequestOptions{}, errors.New("AdUnitID is empty")
	}

	demandData, ok := auctionRequest.AdObject.Demands[adapter.YandexKey]
	if !ok {
		return nil, adapters.RTBRequestOptions{}, errors.New("yandex demand data is missing")
	}

	token, ok := demandData["token"].(string)
	if !ok || token == "" {
		return nil, adapters.RTBRequestOptions{}, errors.New("yandex bidder token is empty")
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
		return nil, adapters.RTBRequestOptions{}, errors.New("unknown impression type")
	}

	imp.Rwdd = rwdd
	impExt, err := json.Marshal(map[string]any{
		"ad_type": adTypeString,
	})
	if err != nil {
		return nil, adapters.RTBRequestOptions{}, err
	}
	imp.Ext = impExt

	return imp, adapters.RTBRequestOptions{TagID: a.AdUnitID}, nil
}

func (a *YandexAdapter) EnrichOpenRTBRequest(request *openrtb.BidRequest, auctionRequest *schema.AuctionRequest) error {
	token, _ := auctionRequest.AdObject.Demands[adapter.YandexKey]["token"].(string)
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
	return nil
}

func (a *YandexAdapter) ExecuteOptions(openrtb.BidRequest) (adapters.ExecuteRTBOptions, error) {
	return adapters.ExecuteRTBOptions{
		URL:   yandexEndpoint,
		TagID: a.AdUnitID,
	}, nil
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
