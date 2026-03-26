package notification

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/bidon-io/bidon-backend/internal/insights"
	"github.com/cenkalti/backoff/v4"
	"github.com/prebid/openrtb/v19/openrtb3"

	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
)

type Params struct {
	Bundle           string
	AdType           string
	AuctionID        string
	NotificationType string
	InsightProvider  string
	URL              string
	Bid              Bid
	Reason           openrtb3.LossReason
	FirstPrice       float64
	SecondPrice      float64
	AuctionWinner    string
	UsedFloor        float64
	ExtraMacros      map[string]string
}

type EventSender struct {
	HttpClient  *http.Client
	EventLogger *event.Logger
}

func (es EventSender) SendEvent(ctx context.Context, p Params) {
	u, err := url.Parse(p.URL)
	if p.URL == "" || err != nil {
		log.Printf("SendNotificationEvent: failed to parse URL type %s: %s", p.NotificationType, p.URL)
		return
	}
	macroses := macrosesMap(
		p.Bid,
		p.Reason,
		p.FirstPrice,
		p.SecondPrice,
		p.AuctionWinner,
		p.UsedFloor,
		p.InsightProvider,
		p.ExtraMacros,
	)
	params, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		log.Printf("SendNotificationEvent: failed to parse params: %s", u.RawQuery)
		return
	}
	for param := range params {
		if val, ok := macroses[params.Get(param)]; ok {
			params.Set(param, val)
		}
	}
	u.RawQuery = params.Encode()
	err = backoff.Retry(func() error {
		httpResp, err := es.HttpClient.Get(u.String())
		if err != nil {
			log.Printf("SendNotificationEvent: send failed: %v", err)
			return err
		}
		defer httpResp.Body.Close()

		return nil
	}, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3))

	e := event.NewNotificationEvent(event.NotificationParams{
		EventType:   p.NotificationType,
		ImpID:       p.Bid.ImpID,
		Bundle:      p.Bundle,
		AdType:      p.AdType,
		AuctionID:   p.AuctionID,
		DemandID:    string(p.Bid.DemandID),
		LossReason:  int64(p.Reason),
		Price:       p.Bid.Price,
		FirstPrice:  p.FirstPrice,
		SecondPrice: p.SecondPrice,
		URL:         u.String(),
		TemplateURL: p.URL,
		Error:       err,
	})
	es.EventLogger.Log(e, func(err error) {
		log.Printf("SendNotificationEvent: log notification event: %v", err)
	})

	if err != nil {
		log.Printf("SendNotificationEvent: failed to send loss notification: %s -> %s", p.Bid.DemandID, p.URL)
	}
}

func macrosesMap(
	bid Bid,
	lossReason openrtb3.LossReason,
	firstPrice, secondPrice float64,
	auctionWinner string,
	usedFloor float64,
	insightProvider string,
	extraMacros map[string]string,
) map[string]string {
	baseMacros := map[string]string{
		"${AUCTION_MIN_TO_WIN}":         strconv.FormatFloat(secondPrice, 'f', -1, 64),
		"${AUCTION_MINIMUM_BID_TO_WIN}": strconv.FormatFloat(secondPrice, 'f', -1, 64),
		"${MIN_BID_TO_WIN}":             strconv.FormatFloat(secondPrice, 'f', -1, 64),
		"${AUCTION_ID}":                 bid.RequestID,
		"${AUCTION_BID_ID}":             bid.ID,
		"${AUCTION_IMP_ID}":             bid.ImpID,
		"${AUCTION_SEAT_ID}":            bid.SeatID,
		"${AUCTION_AD_ID}":              bid.AdID,
		"${AUCTION_PRICE}":              strconv.FormatFloat(firstPrice, 'f', -1, 64),
		"${AUCTION_LOSS}":               fmt.Sprintf("%d", lossReason),
		"${AUCTION_CURRENCY}":           "USD",
		"${AUCTION_WINNER}":             auctionWinner,
		"${AUCTION_USED_FP}":            strconv.FormatFloat(usedFloor, 'f', -1, 64),
	}

	for macro, value := range providerAwareMacros(insightProvider, auctionWinner, firstPrice, usedFloor) {
		baseMacros[macro] = value
	}
	for macro, value := range extraMacros {
		baseMacros[macro] = value
	}

	return baseMacros
}

func providerAwareMacros(insightProvider, auctionWinner string, firstPrice, usedFloor float64) map[string]string {
	normalizedProvider := insights.Key(insightProvider)
	switch normalizedProvider {
	case insights.NeftaKey:
		// Keep provider-specific aliases here, while preserving legacy AUCTION_* macros.
		// Nefta spec requires float precision of up to 6 decimal places.
		return map[string]string{
			"${AUCTION_PRICE}":   strconv.FormatFloat(firstPrice, 'f', 6, 64),
			"${AUCTION_USED_FP}": strconv.FormatFloat(usedFloor, 'f', 6, 64),
			"${WINNER}":          auctionWinner,
			"${USED_FP}":         strconv.FormatFloat(usedFloor, 'f', 6, 64),
		}
	default:
		return nil
	}
}
