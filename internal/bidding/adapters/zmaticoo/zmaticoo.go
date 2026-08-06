package zmaticoo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bidon-io/bidon-backend/internal/bidding/adapters/geo"
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

var _ adapters.BidderInterface = (*ZmaticooAdapter)(nil)

func (a *ZmaticooAdapter) banner(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	imp, _ := adapters.BuildBannerImp(auctionRequest, adapters.BannerImpOptions{})
	return imp
}

func (a *ZmaticooAdapter) interstitial(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	return adapters.BuildInterstitialImp(auctionRequest, adapters.InterstitialImpOptions{
		DisableOrientationSwap: true,
	})
}

func (a *ZmaticooAdapter) rewarded(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	return adapters.BuildRewardedImp(auctionRequest, adapters.RewardedImpOptions{
		DisableOrientationSwap: true,
	})
}

func (a *ZmaticooAdapter) BuildImpression(request openrtb.BidRequest, auctionRequest *schema.AuctionRequest) (*openrtb2.Imp, adapters.RTBRequestOptions, error) {
	if a.PlacementID == "" {
		return nil, adapters.RTBRequestOptions{}, errors.New("PlacementID is empty")
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
		TagID: a.PlacementID,
	}
	if request.App != nil {
		opts.AppID = a.AppID
	}

	return imp, opts, nil
}

func (a *ZmaticooAdapter) EnrichOpenRTBRequest(request *openrtb.BidRequest, auctionRequest *schema.AuctionRequest) error {
	extJSON, err := a.buildRequestExt(auctionRequest)
	if err != nil {
		return err
	}
	request.Ext = extJSON
	return nil
}

func (a *ZmaticooAdapter) ExecuteRequest(ctx context.Context, client *http.Client, request openrtb.BidRequest) *adapters.DemandResponse {
	impID := ""
	// Zmaticoo response is not OpenRTB, so we store imp ID for fallback in ParseBids.
	if len(request.Imp) > 0 {
		impID = request.Imp[0].ID
	}

	return adapters.ExecuteRTBRequest(ctx, client, request, adapters.ExecuteRTBOptions{
		DemandID:    adapter.ZmaticooKey,
		URL:         getEndpoint(adapters.CountryFromRequest(request)),
		TagID:       a.PlacementID,
		PlacementID: a.PlacementID,
		ImpID:       impID,
		Headers:     http.Header{"X-OpenRTB-Version": {"2.5"}},
	})
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

	dr.Bid = &adapters.DemandBid{
		ID:       br.RequestID,
		ImpID:    impID, // Zmaticoo response is not OpenRTB, so we get it from request
		Price:    br.ECPM,
		Payload:  br.RequestID, // Zmaticoo response is not OpenRTB, so we get br.RequestID
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
