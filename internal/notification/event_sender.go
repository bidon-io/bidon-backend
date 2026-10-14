package notification

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/cenkalti/backoff/v4"
	"github.com/prebid/openrtb/v19/openrtb3"
	"go.uber.org/zap"

	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
)

type Params struct {
	Bundle           string
	AdType           string
	AuctionID        string
	NotificationType string
	URL              string
	Bid              Bid
	Reason           openrtb3.LossReason
	FirstPrice       float64
	SecondPrice      float64
}

type EventSender struct {
	HttpClient  *http.Client
	EventLogger *event.Logger
	Logger      *zap.Logger
}

func (es EventSender) log() *zap.Logger {
	if es.Logger == nil {
		return zap.NewNop()
	}
	return es.Logger
}

func (es EventSender) SendEvent(ctx context.Context, p Params) {
	u, err := url.Parse(p.URL)
	if p.URL == "" || err != nil {
		log.Printf("SendNotificationEvent: failed to parse URL type %s: %s", p.NotificationType, p.URL)
		return
	}
	macroses := macrosesMap(p.Bid, p.Reason, p.FirstPrice, p.SecondPrice)
	params, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		log.Printf("SendNotificationEvent: failed to parse params: %s", u.RawQuery)
		return
	}
	for param := range params {
		if val, ok := macroses[params.Get(param)]; ok {
			params.Set(param, val)
		} else {
			log.Printf("SendNotificationEvent: cannot replace param %s", param)
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
	// sendErr is the notification HTTP outcome. The produce callback's error
	// is separate: a full Kafka buffer drops the record after the HTTP call.
	sendErr := err
	notifiedURL := u.String()
	es.EventLogger.Log(e, func(produceErr error) {
		es.logDropped(p, notifiedURL, sendErr, produceErr)
	})

	if err != nil {
		log.Printf("SendNotificationEvent: failed to send loss notification: %s -> %s", p.Bid.DemandID, p.URL)
	}
}

// logDropped records a notification that was not written to Kafka, with
// enough fields to reconstruct which notice was lost.
func (es EventSender) logDropped(p Params, notifiedURL string, sendErr, produceErr error) {
	if produceErr == nil {
		return
	}
	outcome := "ok"
	fields := []zap.Field{
		zap.String("event_type", p.NotificationType),
		zap.String("auction_id", p.AuctionID),
		zap.String("demand_id", string(p.Bid.DemandID)),
		zap.String("imp_id", p.Bid.ImpID),
		zap.String("bundle", p.Bundle),
		zap.String("ad_type", p.AdType),
		zap.String("url", notifiedURL),
		zap.String("template_url", p.URL),
		zap.NamedError("produce_error", produceErr),
	}
	if sendErr != nil {
		outcome = "error"
		fields = append(fields, zap.NamedError("notification_error", sendErr))
	}
	fields = append(fields, zap.String("notification_outcome", outcome))
	es.log().Error("dropped notification event", fields...)
}

func macrosesMap(bid Bid, lossReason openrtb3.LossReason, firstPrice, secondPrice float64) map[string]string {
	return map[string]string{
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
	}
}
