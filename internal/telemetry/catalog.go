package telemetry

import (
	"errors"
	"time"

	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/sdkapi"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	telemetryv1 "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/telemetry/v1"
)

// Params is the request context shared by every catalog emit.
type Params struct {
	Request *schema.AuctionRequest
	App     *sdkapi.App
	Country string
	TraceID string
}

type AuctionCompletedParams struct {
	Params
	Bids    []adapters.DemandResponse
	Started time.Time
	Err     error
}

func (e Event) AuctionRequestReceived(params Params) {
	req := params.Request
	priceFloor := 0.0
	if req != nil {
		priceFloor = req.AdObject.PriceFloor
	}
	a := attrsFrom(params)
	e.emit(&telemetryv1.AuctionRequestReceived{
		Envelope:   newProtoEnvelope(a, EventAuctionRequestReceived),
		PriceFloor: priceFloor,
	}, EventAuctionRequestReceived, "", a)
}

func (e Event) AuctionCompleted(params AuctionCompletedParams) {
	a := attrsFrom(params.Params)
	ev := &telemetryv1.AuctionCompleted{
		Envelope:         newProtoEnvelope(a, EventAuctionCompleted),
		Scope:            telemetryv1.Scope_SCOPE_BIDDING_ROUND,
		TotalLatencyMs:   time.Since(params.Started).Milliseconds(),
		ParticipantCount: int32(len(params.Bids)),
	}
	if params.Request != nil {
		winner, price := roundWinner(params.Bids, params.Request.AdObject.PriceFloor)
		ev.WinnerDsp = winner
		ev.Price = price
	}
	result := AuctionResultOK
	if params.Err != nil {
		ev.ErrorCode = protoErrorCode(errorCode(params.Err))
		result = AuctionResultError
	}
	e.emit(ev, EventAuctionCompleted, "", a)
	ObserveAuctionCompleted(result)
}

func (e Event) DSPRequestSent(params Params, dsp string) {
	a := attrsFrom(params)
	e.emit(&telemetryv1.DspRequestSent{
		Envelope: newProtoEnvelope(a, EventDSPRequestSent),
		Scope:    telemetryv1.Scope_SCOPE_BIDDING_ROUND,
		Dsp:      dsp,
	}, EventDSPRequestSent, dsp, a)
}

func (e Event) DSPResponseReceived(params Params, dr *adapters.DemandResponse) {
	if dr == nil {
		return
	}
	a := attrsFrom(params)
	latencyMS := dr.EndTS - dr.StartTS
	outcome := OutcomeFromDemand(dr.Error, dr.IsBid(), dr.Status)
	ev := &telemetryv1.DspResponseReceived{
		Envelope:   newProtoEnvelope(a, EventDSPResponseReceived),
		Scope:      telemetryv1.Scope_SCOPE_BIDDING_ROUND,
		Dsp:        string(dr.DemandID),
		Outcome:    protoOutcome(outcome),
		HttpStatus: int32(dr.Status),
		LatencyMs:  latencyMS,
	}
	if dr.IsBid() {
		ev.Price = dr.Price()
	}
	e.emit(ev, EventDSPResponseReceived, ev.Dsp, a)
	ObserveDSP(string(dr.DemandID), outcome, float64(latencyMS)/1000)

	e.dspResponseRejectedIfBelowFloor(params, dr)
}

func (e Event) dspResponseRejectedIfBelowFloor(params Params, dr *adapters.DemandResponse) {
	if dr == nil || !dr.IsBid() || params.Request == nil {
		return
	}
	floor := params.Request.AdObject.GetBidFloorForBidding()
	if dr.Price() >= floor {
		return
	}
	a := attrsFrom(params)
	e.emit(&telemetryv1.DspResponseRejected{
		Envelope:     newProtoEnvelope(a, EventDSPResponseRejected),
		Scope:        telemetryv1.Scope_SCOPE_BIDDING_ROUND,
		Dsp:          string(dr.DemandID),
		Price:        dr.Price(),
		PriceFloor:   floor,
		RejectReason: telemetryv1.RejectReason_REJECT_REASON_BELOW_FLOOR,
	}, EventDSPResponseRejected, string(dr.DemandID), a)
}

func attrsFrom(params Params) attrs {
	var appID int64
	if params.App != nil {
		appID = params.App.ID
	}
	req := params.Request
	if req == nil {
		return attrs{AppID: appID, Country: params.Country, TraceID: params.TraceID}
	}
	return attrs{
		AppID:     appID,
		AuctionID: req.AdObject.AuctionID,
		SessionID: req.Session.ID,
		AdType:    string(req.AdType),
		AdFormat:  string(req.AdObject.Format()),
		Country:   params.Country,
		TraceID:   params.TraceID,
	}
}

func roundWinner(bids []adapters.DemandResponse, floor float64) (string, float64) {
	var winner string
	var price float64
	for _, bid := range bids {
		if !bid.IsBid() || bid.Price() <= floor {
			continue
		}
		if bid.Price() > price {
			winner = string(bid.DemandID)
			price = bid.Price()
		}
	}
	return winner, price
}

func errorCode(err error) ErrorCode {
	switch {
	case errors.Is(err, sdkapi.ErrNoAdsFound):
		return ErrorCodeNoAdsFound
	case errors.Is(err, sdkapi.ErrInvalidAuctionKey):
		return ErrorCodeInvalidAuctionKey
	default:
		return ErrorCodeError
	}
}
