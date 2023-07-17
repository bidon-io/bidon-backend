package bidmachine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	"github.com/prebid/openrtb/v19/adcom1"
	"github.com/prebid/openrtb/v19/openrtb2"
)

type bidmachineAdapter struct {
	sellerID string
	endpoint string
}

type ExtImpBidmachine struct {
	Host     string `json:"host"`
	Path     string `json:"path"`
	SellerID string `json:"seller_id"`
}

var bannerFormats = map[string][2]int64{
	"BANNER":      {320, 50},
	"LEADERBOARD": {728, 90},
	"MREC":        {300, 250},
	"ADAPTIVE":    {0, 50},
	"":            {320, 50}, // Default
}

var fullscreenFormats = map[string][2]int64{
	"PHONE":  {320, 480},
	"TABLET": {768, 1024},
}

func (a *bidmachineAdapter) banner(br *schema.BiddingRequest) *openrtb2.Imp {
	size := bannerFormats[string(br.Imp.Format())]
	w, h := size[0], size[1]
	if !br.Imp.IsPortrait() {
		w, h = h, w
	}
	return &openrtb2.Imp{
		Instl: 0,
		Banner: &openrtb2.Banner{
			W:     &w,
			H:     &h,
			BType: []openrtb2.BannerAdType{},
			BAttr: []adcom1.CreativeAttribute{1, 2, 5, 8, 9, 14, 17},
			Pos:   adcom1.PositionAboveFold.Ptr(),
		},
	}
}

func (a *bidmachineAdapter) interstitial(br *schema.BiddingRequest) *openrtb2.Imp {
	size := fullscreenFormats[string(br.Imp.Format())]
	w, h := size[0], size[1]
	if !br.Imp.IsPortrait() {
		w, h = h, w
	}
	return &openrtb2.Imp{
		Instl: 1,
		Banner: &openrtb2.Banner{
			W:     &w,
			H:     &h,
			BType: []openrtb2.BannerAdType{},
			BAttr: []adcom1.CreativeAttribute{},
			Pos:   adcom1.PositionFullScreen.Ptr(),
		},
		Ext: json.RawMessage(`{"rewarded":1}`),
	}
}

func (a *bidmachineAdapter) rewarded(br *schema.BiddingRequest) *openrtb2.Imp {
	size := fullscreenFormats[string(br.Imp.Format())]
	w, h := size[0], size[1]
	if !br.Imp.IsPortrait() {
		w, h = h, w
	}
	return &openrtb2.Imp{
		Instl: 1,
		Banner: &openrtb2.Banner{
			W:     &w,
			H:     &h,
			BType: []openrtb2.BannerAdType{},
			BAttr: []adcom1.CreativeAttribute{16},
			Pos:   adcom1.PositionFullScreen.Ptr(),
		},
		Ext: json.RawMessage(`{"rewarded":1}`),
	}
}

func (a *bidmachineAdapter) CreateRequest(request openrtb2.BidRequest, br *schema.BiddingRequest) (openrtb2.BidRequest, []error) {
	var errs []error
	secure := int8(1)
	headers := http.Header{}
	headers.Add("Content-Type", "application/json")
	headers.Add("Accept", "application/json")
	headers.Add("X-Openrtb-Version", "2.5")

	var imp *openrtb2.Imp
	switch br.Imp.Type() {
	case ad.BannerType:
		imp = a.banner(br)
	case ad.InterstitialType:
		imp = a.interstitial(br)
	case ad.RewardedType:
		imp = a.rewarded(br)
	default:
		return request, []error{errors.New("unknown impression type")}
	}

	imp.ID = "1"
	imp.DisplayManager = string(adapter.BidmachineKey)
	imp.DisplayManagerVer = "123"
	imp.Secure = &secure
	imp.BidFloor = 0.1

	mapStructure := &map[string]interface{}{}
	_ = json.Unmarshal(imp.Ext, mapStructure)

	(*mapStructure)["bid_token"] = "asd"

	raw, _ := json.Marshal(mapStructure)

	imp.Ext = raw

	request.Imp = []openrtb2.Imp{*imp}

	return request, errs
}

func (a *bidmachineAdapter) ExecuteRequest(ctx context.Context, client *http.Client, request openrtb2.BidRequest) *adapters.DemandResponse {
	dr := &adapters.DemandResponse{
		DemandID: adapter.BidmachineKey,
	}
	requestBody, err := json.Marshal(request)
	if err != nil {
		dr.Error = err
		return dr
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		dr.Error = err
		return dr
	}

	httpResp, err := client.Do(httpReq)
	if err != nil {
		if err == context.DeadlineExceeded {
			// doTimeoutNotification if bidder support, eg FB
		}
		dr.Error = err
		return dr
	}

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		dr.Error = err
		return dr
	}
	defer httpResp.Body.Close()

	fmt.Println(string(respBody))

	return dr
}

func (a *bidmachineAdapter) ParseBids(internalRequest openrtb2.BidResponse) []error {
	var errs []error

	// switch responseData.StatusCode {
	// case http.StatusNoContent:
	// 	return nil, nil
	// case http.StatusServiceUnavailable:
	// 	fallthrough
	// case http.StatusBadRequest:
	// 	fallthrough
	// case http.StatusUnauthorized:
	// 	fallthrough
	// case http.StatusForbidden:
	// 	return nil, []error{&errortypes.BadInput{
	// 		Message: "unexpected status code: " + strconv.Itoa(responseData.StatusCode) + " " + string(responseData.Body),
	// 	}}
	// case http.StatusOK:
	// 	break
	// default:
	// 	return nil, []error{&errortypes.BadServerResponse{
	// 		Message: "unexpected status code: " + strconv.Itoa(responseData.StatusCode) + " " + string(responseData.Body),
	// 	}}
	// }

	// var bidResponse openrtb2.BidResponse
	// err := json.Unmarshal(responseData.Body, &bidResponse)
	// if err != nil {
	// 	return nil, []error{&errortypes.BadServerResponse{
	// 		Message: err.Error(),
	// 	}}
	// }

	// response := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))

	// for _, seatBid := range bidResponse.SeatBid {
	// 	for _, bid := range seatBid.Bid {
	// 		thisBid := bid
	// 		bidType := GetMediaTypeForImp(bid.ImpID, request.Imp)
	// 		if bidType == UndefinedMediaType {
	// 			errs = append(errs, &errortypes.BadServerResponse{
	// 				Message: "ignoring bid id=" + bid.ID + ", request doesn't contain any valid impression with id=" + bid.ImpID,
	// 			})
	// 			continue
	// 		}
	// 		response.Bids = append(response.Bids, &adapters.TypedBid{
	// 			Bid:     &thisBid,
	// 			BidType: bidType,
	// 		})
	// 	}
	// }

	return errs
}

// Builder builds a new instance of the Bidmachine adapter for the given bidder with the given config.
func Builder(cfg adapter.Config, client *http.Client) (adapters.Bidder, error) {
	bmCfg := cfg[adapter.BidmachineKey]

	adpt := &bidmachineAdapter{
		endpoint: bmCfg["endpoint"].(string),
		sellerID: bmCfg["seller_id"].(string),
	}

	bidder := adapters.Bidder{
		Adapter: adpt,
		Client:  client,
	}

	return bidder, nil
}
