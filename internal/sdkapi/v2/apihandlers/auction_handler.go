package apihandlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/bidon-io/bidon-backend/internal/auction"
	"github.com/bidon-io/bidon-backend/internal/sdkapi"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

type AuctionHandler struct {
	*BaseHandler[schema.AuctionRequest, *schema.AuctionRequest]
	AuctionService AuctionService
}

type AuctionService interface {
	Run(ctx context.Context, params *auction.ExecutionParams) (*auction.Response, error)
}

type AuctionResponse struct {
	ConfigID                 int64            `json:"auction_configuration_id"`
	ConfigUID                string           `json:"auction_configuration_uid"`
	ExternalWinNotifications bool             `json:"external_win_notifications"`
	AdUnits                  []auction.AdUnit `json:"ad_units"`
	NoBids                   []auction.AdUnit `json:"no_bids"`
	Segment                  auction.Segment  `json:"segment"`
	Token                    string           `json:"token"`
	AuctionPriceFloor        float64          `json:"auction_pricefloor"`
	AuctionTimeout           int              `json:"auction_timeout"`
	AuctionID                string           `json:"auction_id"`
}

func (h *AuctionHandler) Handle(c echo.Context) error {
	req, err := h.resolveRequest(c)
	if err != nil {
		return err
	}

	// Check if we should return empty response for iOS + MAX + specific SDK versions
	if h.shouldReturnEmptyResponse(&req.raw) {
		emptyResponse := h.buildEmptyResponse(&req.raw, req.auctionConfig)
		return c.JSON(http.StatusOK, emptyResponse)
	}

	params := &auction.ExecutionParams{
		Req:     &req.raw,
		AppID:   req.app.ID,
		Country: req.countryCode(),
		GeoData: req.geoData,
		Log: func(str string) {
			c.Logger().Printf(str)
		},
		LogErr: func(err error) {
			sdkapi.LogError(c, err)
		},
	}
	result, err := h.AuctionService.Run(c.Request().Context(), params)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}

// shouldReturnEmptyResponse checks if the request matches conditions for returning empty ads array:
// - OS is iOS
// - Mediator is MAX
// - SDK version is 0.7.x or 0.8.1
func (h *AuctionHandler) shouldReturnEmptyResponse(req *schema.AuctionRequest) bool {
	// Check OS is iOS
	if strings.ToLower(req.Device.OS) != "iOS" {
		return false
	}

	// Check mediator is MAX
	if strings.ToLower(req.GetMediator()) != "max" {
		return false
	}

	// Check SDK version
	sdkVersion, err := req.GetSDKVersionSemver()
	if err != nil {
		return false
	}

	// Check if version matches 0.7.x or 0.8.1
	return sdkapi.Version07xConstraint.Check(sdkVersion) || sdkapi.Version081Constraint.Check(sdkVersion)
}

// buildEmptyResponse creates an empty auction response with proper structure
func (h *AuctionHandler) buildEmptyResponse(req *schema.AuctionRequest, auctionConfig *auction.Config) *auction.Response {
	response := &auction.Response{
		AdUnits: make([]auction.AdUnit, 0),
		NoBids:  make([]auction.AdUnit, 0),
		Segment: auction.Segment{
			ID:  req.Segment.ID,
			UID: req.Segment.UID,
		},
		Token:             "{}",
		AuctionID:         req.AdObject.AuctionID,
		AuctionPriceFloor: req.AdObject.PriceFloor,
	}

	// Set auction configuration data if available
	if auctionConfig != nil {
		response.ConfigID = auctionConfig.ID
		response.ConfigUID = auctionConfig.UID
		response.ExternalWinNotifications = auctionConfig.ExternalWinNotifications
		response.AuctionTimeout = auctionConfig.Timeout
	}

	return response
}
