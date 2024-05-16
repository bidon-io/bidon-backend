package sdkapi

import (
	"errors"
	"fmt"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/auction"
	"github.com/bidon-io/bidon-backend/internal/hybrid_auction"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	"github.com/bidon-io/bidon-backend/internal/segment"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
)

type HybridAuctionHandler struct {
	*BaseHandler[schema.HybridAuctionRequest, *schema.HybridAuctionRequest]
	HybridAuctionBuilder  *hybrid_auction.Builder
	SegmentMatcher        *segment.Matcher
	BiddingBuilder        BiddingBuilder
	AdUnitsMatcher        AdUnitsMatcher
	AdaptersConfigBuilder AdaptersConfigBuilder
	EventLogger           *event.Logger
}

type HybridAuctionResponse struct {
	ConfigID                 int64            `json:"auction_configuration_id"`
	ConfigUID                string           `json:"auction_configuration_uid"`
	ExternalWinNotifications bool             `json:"external_win_notifications"`
	AdUnits                  []auction.AdUnit `json:"ad_units"`
	Bids                     []Bid            `json:"bids,omitempty"`
	Segment                  auction.Segment  `json:"segment"`
	Token                    string           `json:"token"`
	PriceFloor               float64          `json:"pricefloor"`
	AuctionID                string           `json:"auction_id"`
}

func (h *HybridAuctionHandler) Handle(c echo.Context) error {
	req, err := h.resolveRequest(c)
	if err != nil {
		return err
	}

	segmentParams := &segment.Params{
		Country: req.countryCode(),
		Ext:     req.raw.Segment.Ext,
		AppID:   req.app.ID,
	}

	sgmnt := h.SegmentMatcher.Match(c.Request().Context(), segmentParams)
	req.raw.Segment.ID = sgmnt.StringID()
	req.raw.Segment.UID = sgmnt.UID

	params := &hybrid_auction.BuildParams{
		AppID:                req.app.ID,
		AdType:               req.raw.AdType,
		AdFormat:             req.raw.HybridImp.Format(),
		DeviceType:           req.raw.Device.Type,
		Adapters:             req.raw.Adapters.Keys(),
		Segment:              sgmnt,
		PriceFloor:           req.raw.HybridImp.PriceFloor,
		MergedAuctionRequest: &req.raw,
		GeoData:              req.geoData,
	}

	auctionResult, err := h.HybridAuctionBuilder.Build(c.Request().Context(), params)
	if err != nil {
		if errors.Is(err, auction.ErrNoAdsFound) {
			err = ErrNoAdsFound
		}

		return err
	}
	c.Logger().Printf("[HYBRID AUCTION] auction: (%+v), err: (%s), took (%ms)", auctionResult, err, auctionResult.Stat.DurationTS)

	adUnitsMap := make(map[adapter.Key][]auction.AdUnit)
	for _, adUnit := range *auctionResult.AdUnits {
		key := adapter.Key(adUnit.DemandID)
		adUnitsMap[key] = append(adUnitsMap[key], adUnit)
	}

	h.logEvents(c, req, auctionResult, &adUnitsMap)

	response, err := h.buildResponse(req, auctionResult, &adUnitsMap)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, response)
}

func (h *HybridAuctionHandler) buildResponse(
	req *request[schema.HybridAuctionRequest, *schema.HybridAuctionRequest],
	auctionResult *hybrid_auction.AuctionResult,
	adUnitsMap *map[adapter.Key][]auction.AdUnit,
) (*HybridAuctionResponse, error) {
	imp := req.raw.HybridImp
	response := HybridAuctionResponse{
		ConfigID:   auctionResult.AuctionConfiguration.ID,
		ConfigUID:  auctionResult.AuctionConfiguration.UID,
		Segment:    auction.Segment{ID: req.raw.Segment.ID, UID: req.raw.Segment.UID},
		Token:      "{}",
		AuctionID:  imp.AuctionID,
		PriceFloor: imp.PriceFloor,
	}

	for _, bidResponse := range auctionResult.BiddingAuctionResult.Bids {
		if bidResponse.IsBid() && bidResponse.Price() >= imp.PriceFloor {
			bid := buildBid(bidResponse, adUnitsMap)
			response.Bids = append(response.Bids, *bid)
		}
	}

	minBidFloor := response.Bids[0].Price
	for _, bid := range response.Bids {
		if bid.Price < minBidFloor {
			minBidFloor = bid.Price
		}
	}

	if minBidFloor > imp.PriceFloor {
		response.PriceFloor = minBidFloor
	}

	for _, adUnit := range *auctionResult.AdUnits {
		if adUnit.GetPriceFloor() >= response.PriceFloor {
			response.AdUnits = append(response.AdUnits, adUnit)
		}
	}

	return &response, nil
}

func (h *HybridAuctionHandler) logEvents(
	c echo.Context,
	req *request[schema.HybridAuctionRequest, *schema.HybridAuctionRequest],
	auctionResult *hybrid_auction.AuctionResult,
	adUnitsMap *map[adapter.Key][]auction.AdUnit,
) {
	auctionRequest := &request[schema.AuctionRequest, *schema.AuctionRequest]{
		raw:           req.raw.ToAuctionRequest(),
		app:           req.app,
		auctionConfig: req.auctionConfig,
		geoData:       req.geoData,
	}
	auc := &auction.Auction{
		ConfigID:  auctionResult.AuctionConfiguration.ID,
		ConfigUID: auctionResult.AuctionConfiguration.UID,
	}
	auctionConfigurationUID, err := strconv.Atoi(auc.ConfigUID)
	if err != nil {
		auctionConfigurationUID = 0
	}

	var firstRoundID string
	if len(auctionResult.AuctionConfiguration.Rounds) > 0 {
		firstRoundID = auctionResult.AuctionConfiguration.Rounds[0].ID
	}
	biddingRequest := &request[schema.BiddingRequest, *schema.BiddingRequest]{
		raw:           req.raw.ToBiddingRequest(firstRoundID),
		app:           req.app,
		auctionConfig: req.auctionConfig,
		geoData:       req.geoData,
	}
	events := prepareBiddingEvents(biddingRequest, auctionResult.BiddingAuctionResult, adUnitsMap)
	aucRequestEvent := prepareAuctionRequestEvent(auctionRequest, auc, auctionConfigurationUID)

	events = append(events, aucRequestEvent)
	for _, ev := range events {
		h.EventLogger.Log(ev, func(err error) {
			logError(c, fmt.Errorf("log %v event: %v", ev.EventType, err))
		})
	}
}
