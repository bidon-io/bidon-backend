package adikteev

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
	"github.com/gofrs/uuid/v5"
	"github.com/prebid/openrtb/v19/adcom1"
	"github.com/prebid/openrtb/v19/openrtb2"
)

type AdikteevAdapter struct {
}

//Loïc Anton: map sdk request ad types to ad formats
//  banner => openrtb2.Banner with MRAID api
//  interstitial and rewarded => openrtb2.banner with MRAID API + openrtb2.video

var bannerFormats = map[ad.Format][2]int64{
	ad.BannerFormat: {320, 50},
	ad.MRECFormat:   {300, 250},
}

var MRAIDAPI = []adcom1.APIFramework{adcom1.APIMRAID10, adcom1.APIMRAID20}

func (a *AdikteevAdapter) banner(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
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

func (a *AdikteevAdapter) interstitial(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	size, found := adapters.FullscreenFormats[string(auctionRequest.Device.Type)]
	if !found {
		return nil
	}
	w, h := size[0], size[1]
	if !auctionRequest.AdObject.IsPortrait() {
		w, h = h, w
	}
	return &openrtb2.Imp{
		Instl: 1, //they don't check the instl field to interpret the ad format, but only look at w and h
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
			Protocols: []adcom1.MediaCreativeSubtype{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
	}
}

func (a *AdikteevAdapter) rewarded(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	return a.interstitial(auctionRequest)
}

func (a *AdikteevAdapter) sdkInstanceID(auctionRequest *schema.AuctionRequest) []byte {
	extStructure := map[string]interface{}{
		"sdkinstanceid": auctionRequest.AdObject.Demands[adapter.AdikteevKey]["token"],
	}
	raw, _ := json.Marshal(extStructure)
	return raw
}

func getEndpoint() string {
	return "http://appodeal-eu.dsp.adikteev.com" //?debug=true"
}

func (a *AdikteevAdapter) CreateRequest(request openrtb.BidRequest, auctionRequest *schema.AuctionRequest) (openrtb.BidRequest, error) {
	secure := int8(1)

	var imp *openrtb2.Imp
	switch auctionRequest.AdObject.Type() {
	case ad.BannerType:
		imp = a.banner(auctionRequest)
	case ad.InterstitialType:
		imp = a.interstitial(auctionRequest)
	case ad.RewardedType:
		imp = a.rewarded(auctionRequest)
	default:
		return request, errors.New("unknown impression type")
	}

	impId, _ := uuid.NewV4()
	imp.ID = impId.String()
	imp.DisplayManager = string(adapter.AdikteevKey)
	imp.DisplayManagerVer = auctionRequest.Adapters[adapter.AdikteevKey].SDKVersion
	imp.Secure = &secure
	imp.BidFloor = adapters.CalculatePriceFloor(&request, auctionRequest)

	request.App.Ext = a.sdkInstanceID(auctionRequest)

	request.Imp = []openrtb2.Imp{*imp}
	request.Cur = []string{"USD"}

	return request, nil
}

func (a *AdikteevAdapter) ExecuteRequest(ctx context.Context, client *http.Client, request openrtb.BidRequest) *adapters.DemandResponse {
	dr := &adapters.DemandResponse{
		DemandID:  adapter.AdikteevKey,
		RequestID: request.ID,
	}
	requestBody, err := json.Marshal(request)
	if err != nil {
		dr.Error = err
		return dr
	}
	dr.RawRequest = string(requestBody)

	url := getEndpoint()
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

	return dr
}

// Builder builds a new instance of the Bidmachine adapter for the given bidder with the given config.
func Builder(cfg adapter.ProcessedConfigsMap, client *http.Client) (*adapters.Bidder, error) {
	adpt := &AdikteevAdapter{}

	bidder := &adapters.Bidder{
		Adapter: adpt,
		Client:  client,
	}

	return bidder, nil
}
