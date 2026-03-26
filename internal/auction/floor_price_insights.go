package auction

import (
	"context"
	"strconv"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/insights"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

type floorPriceDecision struct {
	EffectiveFloor        float64
	InsightsNotifications []schema.InsightsNotifications
}

func (s *Service) applyInsightsFloorPrice(
	ctx context.Context,
	params *ExecutionParams,
	adapterKeys []adapter.Key,
	existingFloor float64,
) floorPriceDecision {
	decision := floorPriceDecision{EffectiveFloor: existingFloor}

	if s.InsightsService == nil {
		return decision
	}
	if params == nil || params.Req == nil || params.App == nil {
		return decision
	}

	initReq := insights.InitRequestFromAuctionRequestWithGeoData(params.App.ID, params.Req, params.GeoData)
	bidders := make([]string, len(adapterKeys))
	for i, key := range adapterKeys {
		bidders[i] = string(key)
	}

	floorReq := insights.FloorPriceRequest{
		AppID:                   params.App.ID,
		AuctionID:               params.Req.AdObject.AuctionID,
		AuctionConfigurationID:  params.Req.AdObject.AuctionConfigurationID,
		AuctionConfigurationUID: parseAuctionConfigurationUID(params.Req.AdObject.AuctionConfigurationUID),
		BaseRequest:             initReq.BaseRequest,
		GeoData:                 params.GeoData,
		IDFA:                    initReq.IDFA,
		IDG:                     initReq.IDG,
		IDFV:                    initReq.IDFV,
		AppVersion:              initReq.AppVersion,
		SDKVersion:              initReq.SDKVersion,
		OpenRTB:                 initReq.OpenRTB,
		AdType:                  string(params.Req.AdType),
		AdFormat:                string(params.Req.AdObject.Format()),
		Settings:                params.App.Settings,
		FloorPrice:              existingFloor,
		Bidders:                 bidders,
	}

	results := s.InsightsService.FloorPrice(ctx, floorReq)
	recommendedFloor := existingFloor
	collectedNotifications := make([]schema.InsightsNotifications, 0, len(results))
	for _, result := range results {
		if result.Auction == nil {
			continue
		}
		recommendation := *result.Auction

		// Always collect notification links — even for control group sessions,
		// because Nefta requires all notifications to be fired for uplift calculation.
		if recommendation.Notification.Auction != "" ||
			recommendation.Notification.Impression != "" ||
			recommendation.Notification.Click != "" {
			collectedNotifications = append(collectedNotifications, schema.InsightsNotifications{
				InsightProvider: string(result.Provider),
				Auction:         recommendation.Notification.Auction,
				Impression:      recommendation.Notification.Impression,
				Click:           recommendation.Notification.Click,
			})
		}

		if recommendation.FloorPrice > recommendedFloor {
			recommendedFloor = recommendation.FloorPrice
		}
	}

	decision.InsightsNotifications = collectedNotifications
	decision.EffectiveFloor = recommendedFloor
	return decision
}

func parseAuctionConfigurationUID(uid string) int64 {
	value, err := strconv.ParseInt(uid, 10, 64)
	if err != nil {
		return 0
	}
	return value
}
