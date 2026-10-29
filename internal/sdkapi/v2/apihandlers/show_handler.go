package apihandlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/bidon-io/bidon-backend/internal/db"
	"github.com/bidon-io/bidon-backend/internal/sdkapi"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/util"
	"github.com/bidon-io/bidon-backend/internal/telemetry"
)

type ShowHandler struct {
	*BaseHandler[schema.ShowRequest, *schema.ShowRequest]
	EventLogger         *event.Logger
	NotificationHandler ShowNotificationHandler
	AdUnitLookup        AdUnitLookup
	Telemetry           *telemetry.Logger
}

//go:generate go run -mod=mod github.com/matryer/moq@v0.5.3 -out mocks/show_mocks.go -pkg mocks . ShowNotificationHandler AdUnitLookup
type ShowNotificationHandler interface {
	HandleShow(context.Context, *schema.Bid, string, string)
}

type AdUnitLookup interface {
	GetByUIDCached(context.Context, string) (*db.LineItem, error)
}

func (h *ShowHandler) Handle(c echo.Context) error {
	req, err := h.resolveRequest(c)
	if err != nil {
		return err
	}

	demandRequestEvent := prepareShowEvent(c.Request().Context(), req, h.AdUnitLookup)
	h.EventLogger.Log(demandRequestEvent, func(err error) {
		sdkapi.LogError(c, fmt.Errorf("log show event: %v", err))
	})

	bid := req.raw.Bid
	format := ""
	if bid != nil {
		format = string(bid.Format())
	}
	ctx := withNoticeMeta(c.Request().Context(), req.app, req.raw.BaseRequest, req.countryCode(), string(req.raw.AdType), format)
	if h.Telemetry != nil && bid != nil {
		// Synthesized until the SDK sends ad_impression on this call.
		// AdImpression publishes a client row as-is when its event_id is set,
		// and does not emit a second one.
		h.Telemetry.Event().AdImpression(telemetry.Identity{
			AppID:     req.app.ID,
			AuctionID: bid.AuctionID,
			SessionID: req.raw.Session.ID,
			AdType:    string(req.raw.AdType),
			AdFormat:  format,
			Country:   req.countryCode(),
		}, bid.DemandID, bid.GetPrice(), nil)
	}

	h.NotificationHandler.HandleShow(ctx, bid, req.raw.App.Bundle, string(req.raw.AdType))

	return c.JSON(http.StatusOK, map[string]any{"success": true})
}

func prepareShowEvent(ctx context.Context, req *request[schema.ShowRequest, *schema.ShowRequest], adUnitLookup AdUnitLookup) *event.AdEvent {
	bid := req.raw.Bid

	auctionConfigurationUID, err := strconv.ParseInt(bid.AuctionConfigurationUID, 10, 64)
	if err != nil {
		auctionConfigurationUID = 0
	}

	var adUnitInternalID int64
	var adUnitCredentials map[string]string

	adUnit, err := adUnitLookup.GetByUIDCached(ctx, bid.AdUnitUID)
	if err == nil && adUnit != nil {
		adUnitInternalID = adUnit.ID
		adUnitCredentials = util.ConvertToStringMap(adUnit.Extra)
	}

	adRequestParams := event.AdRequestParams{
		EventType:               "show",
		AdType:                  string(req.raw.AdType),
		AdFormat:                string(bid.Format()),
		AuctionID:               bid.AuctionID,
		AuctionConfigurationID:  bid.AuctionConfigurationID,
		AuctionConfigurationUID: auctionConfigurationUID,
		Status:                  "",
		RoundID:                 bid.RoundID,
		RoundNumber:             bid.RoundIndex,
		ImpID:                   bid.ImpID,
		DemandID:                bid.DemandID,
		AdUnitUID:               int64(bid.GetAdUnitUID()),
		AdUnitInternalID:        adUnitInternalID,
		AdUnitLabel:             bid.AdUnitLabel,
		AdUnitCredentials:       adUnitCredentials,
		ECPM:                    bid.GetPrice(),
		PriceFloor:              bid.AuctionPriceFloor,
		Bidding:                 bid.IsBidding(),
	}

	return event.NewAdEvent(&req.raw.BaseRequest, adRequestParams, req.geoData)
}
