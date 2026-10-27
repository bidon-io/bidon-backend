package telemetry

import (
	telemetryv1 "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/telemetry/v1"
)

// NoticeOutcome is one notice HTTP attempt after retries.
type NoticeOutcome struct {
	Dsp        string
	Price      float64
	LossReason int32
	Success    bool
	HTTPStatus int32
	RetryCount int32
	LatencyMs  int64
	Attempts   int32
	ErrorCode  string
	NoticeType telemetryv1.NoticeType
}

// AdImpression records a show.
// A client event with an event_id is published as that row. Otherwise /v2/show
// synthesizes one (render_duration_ms and om_enabled unset). The two are not
// both emitted for the same call.
func (e Event) AdImpression(id Identity, dsp string, price float64, client *telemetryv1.AdImpression) {
	if client.GetEnvelope().GetEventId() != "" {
		if dsp == "" {
			dsp = client.GetDsp()
		}
		e.emit(client, dsp, attrsFromEnvelope(client.GetEnvelope()))
		ObserveAdImpression()
		return
	}
	a := id.attrs()
	e.emit(&telemetryv1.AdImpression{
		Envelope: newProtoEnvelope(a),
		Dsp:      dsp,
		Price:    price,
	}, dsp, a)
	ObserveAdImpression()
}

func (e Event) WinNoticeSent(id Identity, o NoticeOutcome) {
	a := id.attrs()
	e.emit(&telemetryv1.WinNoticeSent{
		Envelope:      newProtoEnvelope(a),
		Dsp:           o.Dsp,
		ClearingPrice: o.Price,
		Success:       o.Success,
		HttpStatus:    o.HTTPStatus,
		RetryCount:    o.RetryCount,
		LatencyMs:     o.LatencyMs,
	}, o.Dsp, a)
}

func (e Event) BillingNoticeSent(id Identity, o NoticeOutcome) {
	a := id.attrs()
	e.emit(&telemetryv1.BillingNoticeSent{
		Envelope:      newProtoEnvelope(a),
		Dsp:           o.Dsp,
		BillablePrice: o.Price,
		Success:       o.Success,
		HttpStatus:    o.HTTPStatus,
		RetryCount:    o.RetryCount,
		LatencyMs:     o.LatencyMs,
	}, o.Dsp, a)
}

func (e Event) LossNoticeSent(id Identity, o NoticeOutcome) {
	a := id.attrs()
	e.emit(&telemetryv1.LossNoticeSent{
		Envelope:       newProtoEnvelope(a),
		Dsp:            o.Dsp,
		LossReasonCode: o.LossReason,
		WinningPrice:   o.Price,
		Success:        o.Success,
		HttpStatus:     o.HTTPStatus,
		RetryCount:     o.RetryCount,
		LatencyMs:      o.LatencyMs,
	}, o.Dsp, a)
}

// NoticeDeliveryFailed accompanies the *_sent event when success is false.
func (e Event) NoticeDeliveryFailed(id Identity, o NoticeOutcome) {
	if o.Success {
		return
	}
	a := id.attrs()
	e.emit(&telemetryv1.NoticeDeliveryFailed{
		Envelope:   newProtoEnvelope(a),
		Dsp:        o.Dsp,
		NoticeType: o.NoticeType,
		HttpStatus: o.HTTPStatus,
		Attempts:   o.Attempts,
		LatencyMs:  o.LatencyMs,
		ErrorCode:  o.ErrorCode,
	}, o.Dsp, a)
}

func attrsFromEnvelope(env *telemetryv1.Envelope) attrs {
	if env == nil {
		return attrs{}
	}
	return attrs{
		AppID:     env.GetAppId(),
		AuctionID: env.GetAuctionId(),
		SessionID: env.GetSessionId(),
		AdType:    env.GetAdType(),
		AdFormat:  env.GetAdFormat(),
		Country:   env.GetCountry(),
		TraceID:   env.GetTraceId(),
	}
}
