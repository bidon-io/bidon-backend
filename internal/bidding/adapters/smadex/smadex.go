package smadex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	"github.com/prebid/openrtb/v19/adcom1"
	"github.com/prebid/openrtb/v19/openrtb2"
)

// defaultEndpoint is used when the demand source account has no endpoint configured.
const defaultEndpoint = "https://bon-use1.smadex.com/hyperad/rtb/437617/bid"

type SmadexAdapter struct {
	Endpoint string
}

var bannerFormats = map[ad.Format][2]int64{
	ad.BannerFormat: {320, 50},
	ad.MRECFormat:   {300, 250},
}

var MRAIDAPI = []adcom1.APIFramework{adcom1.APIMRAID20}

func (a *SmadexAdapter) banner(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	size, found := bannerFormats[auctionRequest.AdObject.Format()]
	if !found {
		return nil
	}
	w, h := size[0], size[1]

	return &openrtb2.Imp{
		Instl: 0,
		Banner: &openrtb2.Banner{
			W:   &w,
			H:   &h,
			Pos: adcom1.PositionAboveFold.Ptr(),
			API: MRAIDAPI,
		},
	}
}

func (a *SmadexAdapter) interstitial(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	size, found := adapters.FullscreenFormats[string(auctionRequest.Device.Type)]
	if !found {
		return nil
	}
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
			API: MRAIDAPI,
		},
		Video: &openrtb2.Video{
			W:         w,
			H:         h,
			Pos:       adcom1.PositionFullScreen.Ptr(),
			MIMEs:     []string{"video/mp4"},
			Protocols: []adcom1.MediaCreativeSubtype{adcom1.CreativeVAST40, adcom1.CreativeVAST40Wrapper},
		},
	}
}

func (a *SmadexAdapter) rewarded(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	return a.interstitial(auctionRequest)
}

func (a *SmadexAdapter) sdkInstanceID(auctionRequest *schema.AuctionRequest) []byte {
	extStructure := map[string]interface{}{
		"sdkinstanceid": auctionRequest.AdObject.Demands[adapter.SmadexKey]["token"],
	}
	raw, _ := json.Marshal(extStructure)
	return raw
}

func (a *SmadexAdapter) BuildImpression(_ openrtb.BidRequest, auctionRequest *schema.AuctionRequest) (*openrtb2.Imp, adapters.RTBRequestOptions, error) {
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
	if imp == nil {
		return nil, adapters.RTBRequestOptions{}, errors.New("unknown impression type")
	}

	return imp, adapters.RTBRequestOptions{OmitBidFloorCur: true}, nil
}

func (a *SmadexAdapter) EnrichOpenRTBRequest(request *openrtb.BidRequest, auctionRequest *schema.AuctionRequest) error {
	request.App.Ext = a.sdkInstanceID(auctionRequest)
	return nil
}

func (a *SmadexAdapter) ExecuteRequest(ctx context.Context, client *http.Client, request openrtb.BidRequest) *adapters.DemandResponse {
	dr := &adapters.DemandResponse{
		DemandID:  adapter.SmadexKey,
		RequestID: request.ID,
	}
	requestBody, err := json.Marshal(request)
	if err != nil {
		dr.Error = err
		return dr
	}
	dr.RawRequest = string(requestBody)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.Endpoint, bytes.NewBuffer(requestBody))
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

func Builder(cfg adapter.ProcessedConfigsMap, client *http.Client) (*adapters.Bidder, error) {
	smadexCfg := cfg[adapter.SmadexKey]

	endpoint, ok := smadexCfg["endpoint"].(string)
	if !ok || endpoint == "" {
		endpoint = defaultEndpoint
	}

	adpt := &SmadexAdapter{
		Endpoint: endpoint,
	}

	bidder := &adapters.Bidder{
		Adapter: adpt,
		Client:  client,
	}

	return bidder, nil
}
