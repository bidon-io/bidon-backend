package notification

import (
	"strings"

	"github.com/prebid/openrtb/v19/openrtb3"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

func buildInsightsAuctionNotifications(
	auctionResult *AuctionResult,
	bundle, adType, auctionID string,
	auctionPrice float64,
	auctionWinner string,
	usedFloor float64,
) []Params {
	if auctionResult == nil || len(auctionResult.InsightsNotifications) == 0 {
		return nil
	}

	var winnerBid Bid
	for _, b := range auctionResult.Bids {
		if string(b.DemandID) == auctionWinner && (auctionPrice == 0 || b.Price == auctionPrice) {
			winnerBid = b
			break
		}
	}
	if winnerBid.DemandID == "" && len(auctionResult.Bids) > 0 {
		winnerBid = auctionResult.Bids[0]
	}

	// Per Nefta spec: if the auction was not won, macros should be replaced with zero-length strings.
	var extraMacros map[string]string
	if auctionWinner == "" {
		extraMacros = map[string]string{
			"${AUCTION_PRICE}":   "",
			"${AUCTION_WINNER}":  "",
			"${AUCTION_USED_FP}": "",
			"${WINNER}":          "",
			"${USED_FP}":         "",
		}
	}

	var notifications []Params
	for _, insightsNotification := range auctionResult.InsightsNotifications {
		if insightsNotification.Auction == "" {
			continue
		}
		notifications = append(notifications, Params{
			Bundle:           bundle,
			AdType:           adType,
			AuctionID:        auctionID,
			NotificationType: insightNotificationType("INSIGHTS_AUCTION", insightsNotification.InsightProvider),
			InsightProvider:  insightsNotification.InsightProvider,
			URL:              insightsNotification.Auction,
			Bid:              winnerBid,
			Reason:           openrtb3.LossWon,
			FirstPrice:       auctionPrice,
			SecondPrice:      0,
			AuctionWinner:    auctionWinner,
			UsedFloor:        usedFloor,
			ExtraMacros:      extraMacros,
		})
	}
	return notifications
}

func buildInsightsImpressionNotifications(
	auctionResult *AuctionResult,
	impression *schema.Bid,
	bundle, adType string,
) []Params {
	if impression == nil || auctionResult == nil || len(auctionResult.InsightsNotifications) == 0 {
		return nil
	}

	var notifications []Params
	for _, insightsNotification := range auctionResult.InsightsNotifications {
		if insightsNotification.Impression == "" {
			continue
		}
		notifications = append(notifications, Params{
			Bundle:           bundle,
			AdType:           adType,
			AuctionID:        impression.AuctionID,
			NotificationType: insightNotificationType("INSIGHTS_IMPRESSION", insightsNotification.InsightProvider),
			InsightProvider:  insightsNotification.InsightProvider,
			URL:              insightsNotification.Impression,
			Bid: Bid{
				ImpID:    impression.ImpID,
				Price:    impression.GetPrice(),
				DemandID: adapter.Key(impression.DemandID),
			},
			Reason:        openrtb3.LossWon,
			FirstPrice:    impression.GetPrice(),
			SecondPrice:   0,
			AuctionWinner: impression.DemandID,
			UsedFloor:     impression.AuctionPriceFloor,
		})
	}
	return notifications
}

func buildInsightsClickNotifications(
	auctionResult *AuctionResult,
	bid *schema.Bid,
	bundle, adType string,
) []Params {
	if bid == nil || auctionResult == nil || len(auctionResult.InsightsNotifications) == 0 {
		return nil
	}

	var notifications []Params
	for _, insightsNotification := range auctionResult.InsightsNotifications {
		if insightsNotification.Click == "" {
			continue
		}
		notifications = append(notifications, Params{
			Bundle:           bundle,
			AdType:           adType,
			AuctionID:        bid.AuctionID,
			NotificationType: insightNotificationType("INSIGHTS_CLICK", insightsNotification.InsightProvider),
			InsightProvider:  insightsNotification.InsightProvider,
			URL:              insightsNotification.Click,
			Bid: Bid{
				ImpID:    bid.ImpID,
				Price:    bid.GetPrice(),
				DemandID: adapter.Key(bid.DemandID),
			},
			Reason:        openrtb3.LossWon,
			FirstPrice:    bid.GetPrice(),
			SecondPrice:   0,
			AuctionWinner: bid.DemandID,
			UsedFloor:     bid.AuctionPriceFloor,
		})
	}
	return notifications
}

func insightNotificationType(baseType, insightProvider string) string {
	normalizedProvider := strings.ToUpper(strings.TrimSpace(insightProvider))
	if normalizedProvider == "" {
		return baseType
	}

	return baseType + "_" + normalizedProvider
}
