package telemetry

import (
	"encoding/json"
	"time"

	"github.com/gofrs/uuid/v5"
)

func newEnvelope(a attrs, name string) Envelope {
	id, _ := uuid.NewV4()
	return Envelope{
		EventID:       id.String(),
		EventName:     name,
		EventTS:       time.Now().UnixMilli(),
		SchemaVersion: SchemaVersion,
		AppID:         a.AppID,
		AuctionID:     a.AuctionID,
		SessionID:     a.SessionID,
		AdType:        a.AdType,
		AdFormat:      a.AdFormat,
		Country:       a.Country,
		TraceID:       a.TraceID,
		SamplingRate:  1.0,
	}
}

func marshalCatalog(name string, a attrs, extra any) ([]byte, error) {
	envJSON, err := json.Marshal(newEnvelope(a, name))
	if err != nil {
		return nil, err
	}
	extraJSON, err := json.Marshal(extra)
	if err != nil {
		return nil, err
	}

	var env, extraMap map[string]json.RawMessage
	if err := json.Unmarshal(envJSON, &env); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(extraJSON, &extraMap); err != nil {
		return nil, err
	}
	for k, v := range extraMap {
		env[k] = v
	}
	return json.Marshal(env)
}

type auctionRequestReceived struct {
	attrs
	PriceFloor float64
}

func (auctionRequestReceived) EventName() string { return EventAuctionRequestReceived }

func (e auctionRequestReceived) MarshalJSON() ([]byte, error) {
	return marshalCatalog(e.EventName(), e.attrs, struct {
		PriceFloor float64 `json:"price_floor,omitempty"`
	}{PriceFloor: e.PriceFloor})
}

type auctionCompleted struct {
	attrs
	WinnerDSP        string
	Price            float64
	ParticipantCount int
	TotalLatencyMS   int64
	ErrorCode        ErrorCode
}

func (auctionCompleted) EventName() string { return EventAuctionCompleted }

func (e auctionCompleted) MarshalJSON() ([]byte, error) {
	return marshalCatalog(e.EventName(), e.attrs, struct {
		Scope            Scope     `json:"scope,omitempty"`
		WinnerDSP        string    `json:"winner_dsp,omitempty"`
		Price            float64   `json:"price,omitempty"`
		ParticipantCount int       `json:"participant_count,omitempty"`
		TotalLatencyMS   int64     `json:"total_latency_ms,omitempty"`
		ErrorCode        ErrorCode `json:"error_code,omitempty"`
	}{
		Scope:            ScopeBiddingRound,
		WinnerDSP:        e.WinnerDSP,
		Price:            e.Price,
		ParticipantCount: e.ParticipantCount,
		TotalLatencyMS:   e.TotalLatencyMS,
		ErrorCode:        e.ErrorCode,
	})
}

type dspRequestSent struct {
	attrs
	DSP string
}

func (dspRequestSent) EventName() string { return EventDSPRequestSent }

func (e dspRequestSent) MarshalJSON() ([]byte, error) {
	return marshalCatalog(e.EventName(), e.attrs, struct {
		Scope Scope  `json:"scope,omitempty"`
		DSP   string `json:"dsp,omitempty"`
	}{
		Scope: ScopeBiddingRound,
		DSP:   e.DSP,
	})
}

type dspResponseReceived struct {
	attrs
	DSP        string
	Outcome    Outcome
	HTTPStatus int
	LatencyMS  int64
	Price      float64
}

func (dspResponseReceived) EventName() string { return EventDSPResponseReceived }

func (e dspResponseReceived) MarshalJSON() ([]byte, error) {
	return marshalCatalog(e.EventName(), e.attrs, struct {
		Scope      Scope   `json:"scope,omitempty"`
		DSP        string  `json:"dsp,omitempty"`
		Outcome    Outcome `json:"outcome,omitempty"`
		HTTPStatus int     `json:"http_status,omitempty"`
		LatencyMS  int64   `json:"latency_ms,omitempty"`
		Price      float64 `json:"price,omitempty"`
	}{
		Scope:      ScopeBiddingRound,
		DSP:        e.DSP,
		Outcome:    e.Outcome,
		HTTPStatus: e.HTTPStatus,
		LatencyMS:  e.LatencyMS,
		Price:      e.Price,
	})
}

type dspResponseRejected struct {
	attrs
	DSP        string
	Price      float64
	PriceFloor float64
}

func (dspResponseRejected) EventName() string { return EventDSPResponseRejected }

func (e dspResponseRejected) MarshalJSON() ([]byte, error) {
	return marshalCatalog(e.EventName(), e.attrs, struct {
		Scope        Scope        `json:"scope,omitempty"`
		DSP          string       `json:"dsp,omitempty"`
		Price        float64      `json:"price,omitempty"`
		PriceFloor   float64      `json:"price_floor,omitempty"`
		RejectReason RejectReason `json:"reject_reason,omitempty"`
	}{
		Scope:        ScopeBiddingRound,
		DSP:          e.DSP,
		Price:        e.Price,
		PriceFloor:   e.PriceFloor,
		RejectReason: RejectReasonBelowFloor,
	})
}

// Record is the on-wire decode shape used by tests and the log engine.
type Record struct {
	Envelope
	Scope            Scope        `json:"scope,omitempty"`
	DSP              string       `json:"dsp,omitempty"`
	Outcome          Outcome      `json:"outcome,omitempty"`
	HTTPStatus       int          `json:"http_status,omitempty"`
	LatencyMS        int64        `json:"latency_ms,omitempty"`
	Price            float64      `json:"price,omitempty"`
	PriceFloor       float64      `json:"price_floor,omitempty"`
	WinnerDSP        string       `json:"winner_dsp,omitempty"`
	ParticipantCount int          `json:"participant_count,omitempty"`
	TotalLatencyMS   int64        `json:"total_latency_ms,omitempty"`
	ErrorCode        ErrorCode    `json:"error_code,omitempty"`
	RejectReason     RejectReason `json:"reject_reason,omitempty"`
}
