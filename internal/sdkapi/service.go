package sdkapi

import (
	"context"
	"errors"
	"github.com/bidon-io/bidon-backend/internal/db"
	"github.com/bidon-io/bidon-backend/internal/segment"
	"net/http"

	"github.com/bidon-io/bidon-backend/internal/auction"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	"github.com/labstack/echo/v4"
	"golang.org/x/exp/maps"
)

type Service struct {
	AuctionBuilder *auction.Builder
	AppFetcher     AppFetcher
	SegmentFetcher SegmentFetcher
}

// App represents an app for the purposes of the SDK API
type App struct {
	ID int64
}

var ErrAppNotValid = echo.NewHTTPError(http.StatusUnprocessableEntity, "App is not valid")
var ErrNoAdsFound = echo.NewHTTPError(http.StatusUnprocessableEntity, "No ads found")

type AppFetcher interface {
	Fetch(ctx context.Context, appKey, appBundle string) (*App, error)
}

type SegmentFetcher interface {
	Fetch(ctx context.Context, appID int64) ([]db.Segment, error)
}

type AuctionResponse struct {
	auction.Auction
	Token      string  `json:"token"`
	PriceFloor float64 `json:"pricefloor"`
	AuctionID  string  `json:"auction_id"`
}

func (s *Service) HandleAuction(c echo.Context) error {
	var request schema.Request
	if err := c.Bind(&request); err != nil {
		return err
	}

	ctx := c.Request().Context()

	app, err := s.AppFetcher.Fetch(ctx, request.App.Key, request.App.Bundle)
	if err != nil {
		return err
	}

	segmentParams := &segment.Params{
		Country: request.Geo.Country,
		Ext:     request.Segment.Ext,
	}
	sgmnts, _ := s.SegmentFetcher.Fetch(ctx, app.ID)
	sgmnt := segment.Match(sgmnts, segmentParams)
	var segmentID *int64
	if sgmnt == nil {
		segmentID = nil
	} else {
		segmentID = &sgmnt.ID
	}

	auctionParams := &auction.BuildParams{
		AppID:      app.ID,
		AdType:     request.AdType,
		AdFormat:   request.AdObject.AdFormat(),
		DeviceType: request.Device.Type,
		Adapters:   maps.Keys(request.Adapters),
		SegmentID:  segmentID,
	}
	auc, err := s.AuctionBuilder.Build(ctx, auctionParams)
	if err != nil {
		if errors.Is(err, auction.ErrNoAdsFound) {
			err = ErrNoAdsFound
		}

		return err
	}

	response := &AuctionResponse{
		Auction:    *auc,
		Token:      "{}",
		PriceFloor: request.AdObject.PriceFloor,
		AuctionID:  request.AdObject.AuctionID,
	}

	return c.JSON(http.StatusOK, response)
}
