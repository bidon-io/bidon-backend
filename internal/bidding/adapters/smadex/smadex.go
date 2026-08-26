package smadex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	"github.com/gofrs/uuid/v5"
	"github.com/prebid/openrtb/v19/adcom1"
	"github.com/prebid/openrtb/v19/openrtb2"
)

type SmadexAdapter struct {
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

func getEndpoint() string {
	return "https://bon-use1.smadex.com/hyperad/rtb/437617/bid"
}

func (a *SmadexAdapter) CreateRequest(request openrtb.BidRequest, auctionRequest *schema.AuctionRequest) (openrtb.BidRequest, error) {
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
	imp.DisplayManager = string(adapter.SmadexKey)
	imp.DisplayManagerVer = auctionRequest.Adapters[adapter.SmadexKey].SDKVersion
	imp.Secure = &secure
	imp.BidFloor = adapters.CalculatePriceFloor(&request, auctionRequest)

	request.App.Ext = a.sdkInstanceID(auctionRequest)

	request.Imp = []openrtb2.Imp{*imp}
	request.Cur = []string{"USD"}

	return request, nil
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

func (a *SmadexAdapter) ParseBids(dr *adapters.DemandResponse) (*adapters.DemandResponse, error) {
	switch dr.Status {
	case http.StatusNoContent:
		return dr, nil
	case http.StatusOK:
		break
	default:
		return dr, fmt.Errorf("unexpected status code: %s", strconv.Itoa(dr.Status))
	}

	var bidResponse openrtb2.BidResponse
	err := json.Unmarshal([]byte(dr.RawResponse), &bidResponse)
	if err != nil {
		return dr, err
	}

	seat := bidResponse.SeatBid[0]
	bid := seat.Bid[0]

	dr.Bid = &adapters.BidDemandResponse{
		ID:       bid.ID,
		ImpID:    bid.ImpID,
		Price:    bid.Price,
		Payload:  bid.AdM,
		DemandID: adapter.SmadexKey,
		AdID:     bid.AdID,
		SeatID:   seat.Seat,
		LURL:     bid.LURL,
		NURL:     bid.NURL,
		BURL:     bid.BURL,
	}

	return dr, nil
}

func Builder(cfg adapter.ProcessedConfigsMap, client *http.Client) (*adapters.Bidder, error) {
	adpt := &SmadexAdapter{}

	bidder := &adapters.Bidder{
		Adapter: adpt,
		Client:  client,
	}

	return bidder, nil
}
