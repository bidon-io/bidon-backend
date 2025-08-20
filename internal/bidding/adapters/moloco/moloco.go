package moloco

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

// MolocoAdapter represents the Moloco bidding adapter
type MolocoAdapter struct { //nolint:revive
	TagID    string
	AppID    string
	Endpoint string
}

// bannerFormats defines the supported banner formats and their dimensions
var bannerFormats = map[ad.Format][2]int64{
	ad.BannerFormat:      {320, 50},
	ad.LeaderboardFormat: {728, 90},
	ad.MRECFormat:        {300, 250},
	ad.AdaptiveFormat:    {320, 50},
	ad.EmptyFormat:       {320, 50}, // Default
}

// banner creates a banner impression for the bid request
func (a *MolocoAdapter) banner(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
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

// interstitial creates an interstitial impression for the bid request
func (a *MolocoAdapter) interstitial() *openrtb2.Imp {
	return &openrtb2.Imp{
		Instl: 1,
		Banner: &openrtb2.Banner{
			Pos: adcom1.PositionFullScreen.Ptr(),
		},
	}
}

// rewarded creates a rewarded video impression for the bid request
func (a *MolocoAdapter) rewarded() *openrtb2.Imp {
	return &openrtb2.Imp{
		Instl: 0,
		Video: &openrtb2.Video{
			MIMEs: []string{"video/mp4"},
		},
	}
}

// CreateRequest implements the BidderInterface.CreateRequest method
func (a *MolocoAdapter) CreateRequest(request openrtb.BidRequest, auctionRequest *schema.AuctionRequest) (openrtb.BidRequest, error) {
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

	if a.TagID == "" {
		return request, errors.New("TagID is empty")
	}
	imp.TagID = a.TagID

	imp.DisplayManager = string(adapter.MolocoKey)
	imp.DisplayManagerVer = auctionRequest.Adapters[adapter.MolocoKey].SDKVersion
	imp.Secure = &secure
	imp.BidFloor = adapters.CalculatePriceFloor(&request, auctionRequest)
	imp.BidFloorCur = "USD"

	request.Imp = []openrtb2.Imp{*imp}
	request.Cur = []string{"USD"}

	// Set app ID if configured
	if a.AppID != "" && request.App != nil {
		request.App.ID = a.AppID
	}

	// Add user data if token is available
	if demands, exists := auctionRequest.AdObject.Demands[adapter.MolocoKey]; exists {
		if token, tokenExists := demands["token"]; tokenExists {
			if tokenStr, ok := token.(string); ok && tokenStr != "" {
				request.User = &openrtb.User{
					Data: []openrtb.Data{
						{
							Segment: []openrtb.Segment{
								{
									Signal: tokenStr,
								},
							},
						},
					},
				}
			}
		}
	}

	return request, nil
}

// ExecuteRequest implements the BidderInterface.ExecuteRequest method
func (a *MolocoAdapter) ExecuteRequest(ctx context.Context, client *http.Client, request openrtb.BidRequest) *adapters.DemandResponse {
	dr := &adapters.DemandResponse{
		DemandID:  adapter.MolocoKey,
		RequestID: request.ID,
		TagID:     a.TagID,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		dr.Error = err
		return dr
	}
	dr.RawRequest = string(requestBody)

	// Use configured endpoint or default URL
	url := a.Endpoint
	if url == "" {
		url = "https://api.moloco.com/openrtb/bid"
	}
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

// ParseBids implements the BidderInterface.ParseBids method
func (a *MolocoAdapter) ParseBids(dr *adapters.DemandResponse) (*adapters.DemandResponse, error) {
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

	// Extract signaldata from bid extension if available
	signaldata := ""
	if bid.Ext != nil {
		var extParam map[string]any
		err = json.Unmarshal(bid.Ext, &extParam)
		if err == nil {
			if sd, ok := extParam["signaldata"].(string); ok {
				signaldata = sd
			}
		}
	}

	dr.Bid = &adapters.BidDemandResponse{
		ID:         bid.ID,
		ImpID:      bid.ImpID,
		Price:      bid.Price,
		Payload:    bid.AdM,
		Signaldata: signaldata,
		DemandID:   adapter.MolocoKey,
		AdID:       bid.AdID,
		SeatID:     seat.Seat,
		LURL:       bid.LURL,
		NURL:       bid.NURL,
		BURL:       bid.BURL,
	}

	return dr, nil
}

// Builder builds a new instance of the Moloco adapter for the given bidder with the given config.
func Builder(cfg adapter.ProcessedConfigsMap, client *http.Client) (*adapters.Bidder, error) {
	molocoCfg := cfg[adapter.MolocoKey]

	tagID, ok := molocoCfg["tag_id"].(string)
	if !ok {
		tagID = ""
	}

	appID, ok := molocoCfg["app_id"].(string)
	if !ok {
		appID = ""
	}

	endpoint, ok := molocoCfg["endpoint"].(string)
	if !ok {
		endpoint = ""
	}

	adpt := &MolocoAdapter{
		TagID:    tagID,
		AppID:    appID,
		Endpoint: endpoint,
	}

	bidder := &adapters.Bidder{
		Adapter: adpt,
		Client:  client,
	}

	return bidder, nil
}
