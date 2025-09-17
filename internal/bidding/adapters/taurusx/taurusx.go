package taurusx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gofrs/uuid/v5"
	"github.com/prebid/openrtb/v19/adcom1"
	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

type TaurusXAdapter struct {
	AppID string
	TagID string
}

var bannerFormats = map[ad.Format][2]int64{
	ad.BannerFormat:   {320, 50},
	ad.MRECFormat:     {300, 250},
	ad.AdaptiveFormat: {320, 50},
	ad.EmptyFormat:    {320, 50}, // Default
}

func (a *TaurusXAdapter) banner(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	size, ok := bannerFormats[auctionRequest.AdObject.Format()]
	if !ok {
		size = bannerFormats[ad.EmptyFormat] // Use default
	}

	// Handle adaptive format for tablets
	if auctionRequest.AdObject.IsAdaptive() && auctionRequest.Device.IsTablet() {
		size = [2]int64{728, 90} // Leaderboard format
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

func (a *TaurusXAdapter) interstitial() *openrtb2.Imp {
	w := adapters.FullscreenFormats["PHONE"][0]
	h := adapters.FullscreenFormats["PHONE"][1]
	return &openrtb2.Imp{
		Instl: 1,
		Banner: &openrtb2.Banner{
			W:   &w,
			H:   &h,
			Pos: adcom1.PositionFullScreen.Ptr(),
		},
	}
}

func (a *TaurusXAdapter) rewarded() *openrtb2.Imp {
	w := adapters.FullscreenFormats["PHONE"][0]
	h := adapters.FullscreenFormats["PHONE"][1]
	return &openrtb2.Imp{
		Instl: 1,
		Video: &openrtb2.Video{
			W:           w,
			H:           h,
			Pos:         adcom1.PositionFullScreen.Ptr(),
			MIMEs:       []string{"video/mp4"},
			MinDuration: 15,
			MaxDuration: 30,
			Protocols:   []adcom1.MediaCreativeSubtype{adcom1.CreativeVAST20, adcom1.CreativeVAST30},
		},
	}
}

func (a *TaurusXAdapter) CreateRequest(request openrtb.BidRequest, auctionRequest *schema.AuctionRequest) (openrtb.BidRequest, error) {
	if a.TagID == "" {
		return request, errors.New("TagID is empty")
	}

	secure := int8(1)

	var imp *openrtb2.Imp
	switch auctionRequest.AdObject.Type() {
	case ad.BannerType:
		imp = a.banner(auctionRequest)
	case ad.InterstitialType:
		imp = a.interstitial()
	case ad.RewardedType:
		imp = a.rewarded()
	default:
		return request, errors.New("unknown impression type")
	}

	impID, _ := uuid.NewV4()
	imp.ID = impID.String()
	imp.TagID = a.TagID

	imp.DisplayManager = string(adapter.TaurusXKey)
	imp.DisplayManagerVer = auctionRequest.Adapters[adapter.TaurusXKey].SDKVersion
	imp.Secure = &secure
	imp.BidFloor = adapters.CalculatePriceFloor(&request, auctionRequest)
	imp.BidFloorCur = "USD"

	// Add TaurusX-specific extension with bidding token
	taurusxData := make(map[string]interface{})
	if demandData, ok := auctionRequest.AdObject.Demands[adapter.TaurusXKey]; ok {
		if token, ok := demandData["token"].(string); ok && token != "" {
			taurusxData["bid_token"] = token
		}
	}

	if len(taurusxData) > 0 {
		extStructure := &map[string]interface{}{}
		if imp.Ext != nil {
			_ = json.Unmarshal(imp.Ext, extStructure)
		}
		(*extStructure)["taurusx"] = taurusxData
		raw, _ := json.Marshal(extStructure)
		imp.Ext = raw
	}

	request.Imp = []openrtb2.Imp{*imp}
	request.Cur = []string{"USD"}

	// Set app publisher ID
	if request.App != nil && request.App.Publisher != nil {
		request.App.Publisher.ID = a.AppID
	}

	// Add request-level extension with API key
	reqExt := make(map[string]interface{})
	if request.Ext != nil {
		_ = json.Unmarshal(request.Ext, &reqExt)
	}
	reqExt["api_key"] = a.AppID
	extBytes, _ := json.Marshal(reqExt)
	request.Ext = extBytes

	return request, nil
}

func (a *TaurusXAdapter) ExecuteRequest(ctx context.Context, client *http.Client, request openrtb.BidRequest) *adapters.DemandResponse {
	dr := &adapters.DemandResponse{
		DemandID:  adapter.TaurusXKey,
		RequestID: request.ID,
		TagID:     a.TagID,
	}
	requestBody, err := json.Marshal(request)
	if err != nil {
		dr.Error = err
		return dr
	}
	dr.RawRequest = string(requestBody)

	// TaurusX RTB endpoint - replace with actual endpoint
	url := "https://rtb.taurusx.com/bid"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(requestBody))
	if err != nil {
		dr.Error = err
		return dr
	}
	httpReq.Header.Add("Content-Type", "application/json")
	httpReq.Header.Add("X-OpenRTB-Version", "2.6")

	httpResp, err := client.Do(httpReq)
	if err != nil {
		dr.Error = err
		return dr
	}
	defer httpResp.Body.Close()

	dr.Status = httpResp.StatusCode
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		dr.Error = err
		return dr
	}
	dr.RawResponse = string(respBody)

	parsedDr, err := a.ParseBids(dr)
	if err != nil {
		dr.Error = err
		return dr
	}
	return parsedDr
}

func (a *TaurusXAdapter) ParseBids(dr *adapters.DemandResponse) (*adapters.DemandResponse, error) {
	switch dr.Status {
	case http.StatusNoContent:
		return dr, nil
	case http.StatusServiceUnavailable:
		fallthrough
	case http.StatusBadRequest:
		fallthrough
	case http.StatusUnauthorized:
		fallthrough
	case http.StatusForbidden:
		return dr, fmt.Errorf("unauthorized request: %s", strconv.Itoa(dr.Status))
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

	if len(bidResponse.SeatBid) == 0 || len(bidResponse.SeatBid[0].Bid) == 0 {
		return dr, nil
	}

	seat := bidResponse.SeatBid[0]
	bid := seat.Bid[0]

	dr.Bid = &adapters.BidDemandResponse{
		ID:       bid.ID,
		ImpID:    bid.ImpID,
		Price:    bid.Price,
		Payload:  bid.AdM,
		DemandID: adapter.TaurusXKey,
		AdID:     bid.AdID,
		SeatID:   seat.Seat,
		LURL:     bid.LURL,
		NURL:     bid.NURL,
		BURL:     bid.BURL,
	}

	return dr, nil
}

// Builder builds a new instance of the TaurusX adapter for the given bidder with the given config.
func Builder(cfg adapter.ProcessedConfigsMap, client *http.Client) (*adapters.Bidder, error) {
	tCfg := cfg[adapter.TaurusXKey]

	appID, ok := tCfg["app_id"].(string)
	if !ok || appID == "" {
		return nil, fmt.Errorf("missing app_id param for %s adapter", adapter.TaurusXKey)
	}
	tagID, ok := tCfg["tag_id"].(string)
	if !ok {
		tagID = ""
	}

	adpt := &TaurusXAdapter{
		AppID: appID,
		TagID: tagID,
	}

	bidder := adapters.Bidder{
		Adapter: adpt,
		Client:  client,
	}

	return &bidder, nil
}
