package sdkapi

import (
	"errors"
	"github.com/bidon-io/bidon-backend/internal/geocoder"
	"github.com/bidon-io/bidon-backend/internal/segment"
	"net/http"

	"github.com/bidon-io/bidon-backend/internal/auction"
	"github.com/labstack/echo/v4"
)

type AuctionHandler struct {
	*BaseHandler
	SegmentFetcher SegmentFetcher
	AuctionBuilder *auction.Builder
}

type AuctionResponse struct {
	*auction.Auction
	Token      string  `json:"token"`
	PriceFloor float64 `json:"pricefloor"`
	AuctionID  string  `json:"auction_id"`
}

func (h *AuctionHandler) Handle(c echo.Context) error {
	req, err := h.resolveRequest(c)
	if err != nil {
		return err
	}

	country, _ := geocoder.OfflineGeocoderInstance().FindGeoData(c.Request().Context(), c.RealIP())

	segmentParams := &segment.Params{
		Country: country.CountryCode,
		Ext:     req.raw.Segment.Ext,
	}

	sgmnts, _ := h.SegmentFetcher.Fetch(c.Request().Context(), req.app.ID)
	sgmnt := segment.Match(sgmnts, segmentParams)
	var segmentID *int64
	if sgmnt == nil {
		segmentID = nil
	} else {
		segmentID = &sgmnt.ID
	}

	params := &auction.BuildParams{
		AppID:      req.app.ID,
		AdType:     req.raw.AdType,
		AdFormat:   req.adFormat(),
		DeviceType: req.raw.Device.Type,
		Adapters:   req.adapterKeys(),
		SegmentID:  segmentID,
	}
	auc, err := h.AuctionBuilder.Build(c.Request().Context(), params)
	if err != nil {
		if errors.Is(err, auction.ErrNoAdsFound) {
			err = ErrNoAdsFound
		}

		return err
	}

	response := &AuctionResponse{
		Auction:    auc,
		Token:      "{}",
		PriceFloor: req.raw.AdObject.PriceFloor,
		AuctionID:  req.raw.AdObject.AuctionID,
	}

	return c.JSON(http.StatusOK, response)
}
