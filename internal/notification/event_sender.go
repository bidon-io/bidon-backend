package notification

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/prebid/openrtb/v19/openrtb3"

	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
	"github.com/bidon-io/bidon-backend/internal/telemetry"
	telemetryv1 "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/telemetry/v1"
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
	Meta             telemetry.NoticeMeta
}

type EventSender struct {
	HttpClient  *http.Client
	EventLogger *event.Logger
	Telemetry   *telemetry.Logger
	// retryBackoff replaces the production schedule. Tests in this package set it.
	retryBackoff func() backoff.BackOff
}

var missingNoticeAppID sync.Once

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

	if meta := telemetry.NoticeMetaFrom(ctx); meta.Present() {
		p.Meta = meta
	}
	warnMissingNoticeApp(p.Meta.AppID)

	started := time.Now()
	var attempts int
	var lastStatus int
	err = backoff.Retry(func() error {
		attempts++
		httpResp, getErr := es.HttpClient.Get(u.String())
		if getErr != nil {
			log.Printf("SendNotificationEvent: send failed: %v", getErr)
			return getErr
		}
		defer httpResp.Body.Close()
		_, _ = io.Copy(io.Discard, httpResp.Body)
		lastStatus = httpResp.StatusCode
		if httpResp.StatusCode >= 500 {
			return fmt.Errorf("notice http %d", httpResp.StatusCode)
		}
		if httpResp.StatusCode >= 400 {
			return backoff.Permanent(fmt.Errorf("notice http %d", httpResp.StatusCode))
		}
		return nil
	}, es.backoff())

	result, success, errorCode := classifyNotice(lastStatus, err)
	retryCount := attempts - 1
	if retryCount < 0 {
		retryCount = 0
	}
	outcome := telemetry.NoticeOutcome{
		Dsp:        string(p.Bid.DemandID),
		Price:      p.FirstPrice,
		LossReason: int32(p.Reason),
		Success:    success,
		HTTPStatus: int32(lastStatus),
		RetryCount: int32(retryCount),
		LatencyMs:  time.Since(started).Milliseconds(),
		Attempts:   int32(attempts),
		ErrorCode:  errorCode,
	}
	es.emitNotice(p, outcome)
	telemetry.ObserveNotice(noticeMetricKind(p.NotificationType), result)

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

func (es EventSender) backoff() backoff.BackOff {
	if es.retryBackoff != nil {
		return es.retryBackoff()
	}
	return backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3)
}

func (es EventSender) emitNotice(p Params, o telemetry.NoticeOutcome) {
	adType := p.AdType
	if adType == "" {
		adType = p.Meta.AdType
	}
	id := telemetry.Identity{
		AppID:     p.Meta.AppID,
		AuctionID: p.AuctionID,
		SessionID: p.Meta.SessionID,
		AdType:    adType,
		AdFormat:  p.Meta.AdFormat,
		Country:   p.Meta.Country,
	}
	if p.Meta.HasBase {
		base := p.Meta.Base
		id.Base = &base
	}
	ev := es.Telemetry.Event()
	switch p.NotificationType {
	case "NURL":
		o.NoticeType = telemetryv1.NoticeType_NOTICE_TYPE_WIN
		ev.WinNoticeSent(id, o)
	case "BURL":
		o.NoticeType = telemetryv1.NoticeType_NOTICE_TYPE_BILLING
		ev.BillingNoticeSent(id, o)
	case "TimeoutURL":
		o.NoticeType = telemetryv1.NoticeType_NOTICE_TYPE_TIMEOUT
		ev.LossNoticeSent(id, o)
	default:
		o.NoticeType = telemetryv1.NoticeType_NOTICE_TYPE_LOSS
		ev.LossNoticeSent(id, o)
	}
	ev.NoticeDeliveryFailed(id, o)
}

func noticeMetricKind(notificationType string) string {
	switch notificationType {
	case "NURL":
		return "nurl"
	case "BURL":
		return "burl"
	case "TimeoutURL":
		return "timeout"
	default:
		return "lurl"
	}
}

// classifyNotice maps the last HTTP status to the metric result.
// 2xx succeeds. 5xx has already been retried. 4xx is terminal.
// status 0 is a transport failure.
func classifyNotice(status int, err error) (result string, success bool, errorCode string) {
	switch {
	case status >= 500:
		return "5xx", false, "http_5xx"
	case status >= 400:
		return "4xx", false, "http_4xx"
	case status >= 200 && status < 300:
		return "2xx", true, ""
	case err != nil || status == 0:
		return "timeout", false, "transport"
	default:
		return "4xx", false, "http_4xx"
	}
}

func warnMissingNoticeApp(appID int64) {
	if appID != 0 {
		return
	}
	missingNoticeAppID.Do(func() {
		log.Printf("SendNotificationEvent: app_id missing; notice events will not join on (app_id, auction_id)")
	})
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
