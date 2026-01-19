package zmaticoo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/bidon-io/bidon-backend/internal/bidding/adapters/geo"
	"github.com/gofrs/uuid/v5"
	"github.com/prebid/openrtb/v19/adcom1"
	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

const (
	globalZmaticooEndpoint = "https://sdk-api.maticooads.com/bidRequest"
	zmaticooUSEndpoint     = "https://sdk-api-usa.maticooads.com/bidRequest"
	zmaticooEUEndpoint     = "https://sdk-api-fra.maticooads.com/bidRequest"
)

type ZmaticooAdapter struct {
	AppID       string
	PlacementID string
}

var bannerFormats = map[ad.Format][2]int64{
	ad.BannerFormat:      {320, 50},
	ad.LeaderboardFormat: {728, 90},
	ad.MRECFormat:        {300, 250},
	ad.AdaptiveFormat:    {320, 50},
	ad.EmptyFormat:       {320, 50}, // Default
}

func (a *ZmaticooAdapter) banner(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
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

func (a *ZmaticooAdapter) interstitial(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	size := adapters.FullscreenFormats[string(auctionRequest.Device.Type)]
	w, h := size[0], size[1]
	return &openrtb2.Imp{
		Instl: 1,
		Banner: &openrtb2.Banner{
			W:   &w,
			H:   &h,
			Pos: adcom1.PositionFullScreen.Ptr(),
		},
	}
}

func (a *ZmaticooAdapter) rewarded(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	size := adapters.FullscreenFormats[string(auctionRequest.Device.Type)]
	w, h := size[0], size[1]
	return &openrtb2.Imp{
		Instl: 1,
		Video: &openrtb2.Video{
			W:         w,
			H:         h,
			Pos:       adcom1.PositionFullScreen.Ptr(),
			MIMEs:     []string{"video/mp4", "video/x-m4v", "video/quicktime", "video/mpeg", "video/avi"},
			Protocols: []adcom1.MediaCreativeSubtype{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14},
		},
	}
}

func (a *ZmaticooAdapter) CreateRequest(request openrtb.BidRequest, auctionRequest *schema.AuctionRequest) (openrtb.BidRequest, error) {
	if a.PlacementID == "" {
		return request, errors.New("PlacementID is empty")
	}

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

	impID, _ := uuid.NewV4()
	imp.ID = impID.String()
	imp.TagID = a.PlacementID

	imp.DisplayManager = string(adapter.ZmaticooKey)
	imp.DisplayManagerVer = auctionRequest.Adapters[adapter.ZmaticooKey].SDKVersion
	imp.Secure = &secure
	imp.BidFloor = adapters.CalculatePriceFloor(&request, auctionRequest)
	imp.BidFloorCur = "USD"

	request.Imp = []openrtb2.Imp{*imp}
	request.Cur = []string{"USD"}

	if request.App != nil {
		request.App.ID = a.AppID
	}

	extJSON, err := a.buildRequestExt(auctionRequest)
	if err != nil {
		return request, err
	}
	request.Ext = extJSON

	return request, nil
}

func (a *ZmaticooAdapter) ExecuteRequest(ctx context.Context, client *http.Client, request openrtb.BidRequest) *adapters.DemandResponse {
	dr := &adapters.DemandResponse{
		DemandID:    adapter.ZmaticooKey,
		RequestID:   request.ID,
		TagID:       a.PlacementID,
		PlacementID: a.PlacementID,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		dr.Error = err
		return dr
	}
	dr.RawRequest = string(requestBody)

	// Zmaticoo response is not OpenRTB, so we store imp ID for fallback in ParseBids.
	if len(request.Imp) > 0 {
		dr.ImpID = request.Imp[0].ID
	}

	// Get country code for geographic routing
	alpha3 := ""
	if request.Device != nil && request.Device.Geo != nil {
		alpha3 = request.Device.Geo.Country
	}

	url := getEndpoint(alpha3)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(requestBody))
	if err != nil {
		dr.Error = err
		return dr
	}
	httpReq.Header.Add("Content-Type", "application/json")
	httpReq.Header.Add("X-OpenRTB-Version", "2.5")

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

func (a *ZmaticooAdapter) ParseBids(dr *adapters.DemandResponse) (*adapters.DemandResponse, error) {
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

	// Zmaticoo response is not OpenRTB, so we map its fields manually.
	var response zmaticooBidResponse
	if err := json.Unmarshal([]byte(dr.RawResponse), &response); err != nil {
		return dr, err
	}

	if response.Code != 1 {
		return dr, fmt.Errorf("zmaticoo bid failed: %s (code %d)", response.Msg, response.Code)
	}

	br := response.BidResult

	if br == nil {
		return dr, errors.New("missing bid_result")
	}

	impID := br.ImpID
	if impID == "" {
		impID = dr.ImpID
	}

	dr.Bid = &adapters.BidDemandResponse{
		ID:       br.RequestID,
		ImpID:    impID, // Zmaticoo response is not OpenRTB, so we get it from request
		Price:    br.ECPM,
		Payload:  br.AdM, // Zmaticoo response is not OpenRTB, it will be empty
		DemandID: adapter.ZmaticooKey,
		AdID:     br.AdID, // Zmaticoo response is not OpenRTB, it will be empty
		LURL:     br.LURL,
		NURL:     br.NURL,
		BURL:     br.BURL,
	}

	return dr, nil
}

func (a *ZmaticooAdapter) buildRequestExt(auctionRequest *schema.AuctionRequest) (json.RawMessage, error) {
	ext := map[string]any{
		"adx_id": "bidon",
	}
	sdkToken, timestamp, err := a.extractTokenAndTimestamp(auctionRequest)
	if err != nil {
		return nil, err
	}
	ext["sdk_token"] = sdkToken
	ext["timestamp"] = timestamp

	extJSON, err := json.Marshal(ext)
	if err != nil {
		return nil, err
	}

	return extJSON, nil
}

// Builder builds a new instance of the Zmaticoo adapter for the given bidder with the given config.
func Builder(cfg adapter.ProcessedConfigsMap, client *http.Client) (*adapters.Bidder, error) {
	zmaticooCfg := cfg[adapter.ZmaticooKey]

	appID, ok := zmaticooCfg["app_id"].(string)
	if !ok || appID == "" {
		return nil, fmt.Errorf("missing app_id param for %s adapter", adapter.ZmaticooKey)
	}

	placementID, ok := zmaticooCfg["placement_id"].(string)
	if !ok || placementID == "" {
		return nil, fmt.Errorf("missing placement_id param for %s adapter", adapter.ZmaticooKey)
	}

	adpt := &ZmaticooAdapter{
		AppID:       appID,
		PlacementID: placementID,
	}

	bidder := &adapters.Bidder{
		Adapter: adpt,
		Client:  client,
	}

	return bidder, nil
}

type zmaticooTokenEntry struct {
	Token     string `json:"token"`
	Timestamp int64  `json:"timestamp"`
}

type zmaticooBidResponse struct {
	Code      int                `json:"code"`
	Msg       string             `json:"msg"`
	BidResult *zmaticooBidResult `json:"bid_result"`
}

type zmaticooBidResult struct {
	openrtb2.Bid
	RequestID string  `json:"request_id"`
	ECPM      float64 `json:"ecpm"`
	Exp       int64   `json:"exp"`
}

func (a *ZmaticooAdapter) extractTokenAndTimestamp(auctionRequest *schema.AuctionRequest) (sdkToken string, timestamp int64, err error) {
	token, ok := auctionRequest.AdObject.Demands[adapter.ZmaticooKey]["token"].(string)
	if !ok {
		return sdkToken, timestamp, fmt.Errorf("missing token data for %s adapter", adapter.ZmaticooKey)
	}

	if token == "" {
		return sdkToken, timestamp, fmt.Errorf("empty token data for %s adapter", adapter.ZmaticooKey)
	}

	var placements map[string]zmaticooTokenEntry
	if err := json.Unmarshal([]byte(token), &placements); err != nil {
		return sdkToken, timestamp, fmt.Errorf("invalid token data for %s adapter: %w", adapter.ZmaticooKey, err)
	}

	entry, ok := placements[a.PlacementID]
	if !ok {
		return sdkToken, timestamp, fmt.Errorf("missing token for %s placement %s", adapter.ZmaticooKey, a.PlacementID)
	}

	if entry.Token == "" {
		return sdkToken, timestamp, fmt.Errorf("empty token for %s placement %s", adapter.ZmaticooKey, a.PlacementID)
	}

	if entry.Timestamp == 0 {
		return sdkToken, timestamp, fmt.Errorf("missing timestamp for %s placement %s", adapter.ZmaticooKey, a.PlacementID)
	}

	return entry.Token, entry.Timestamp, nil
}

func getEndpoint(alpha3 string) string {
	switch geo.Region(alpha3) {
	case "eu":
		return zmaticooEUEndpoint
	case "us":
		return zmaticooUSEndpoint
	default:
		return globalZmaticooEndpoint
	}
}
