package telemetry

import (
	"errors"
	"time"

	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/sdkapi"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
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
	ev := auctionRequestReceived{
		attrs:      attrsFrom(params),
		PriceFloor: priceFloor,
	}
	e.emit(ev.EventName(), "", ev.attrs, ev)
}

func (e Event) AuctionCompleted(params AuctionCompletedParams) {
	attrs := attrsFrom(params.Params)
	ev := auctionCompleted{
		attrs:            attrs,
		TotalLatencyMS:   time.Since(params.Started).Milliseconds(),
		ParticipantCount: len(params.Bids),
	}
	if params.Request != nil {
		ev.WinnerDSP, ev.Price = roundWinner(params.Bids, params.Request.AdObject.PriceFloor)
	}
	result := AuctionResultOK
	if params.Err != nil {
		ev.ErrorCode = errorCode(params.Err)
		result = AuctionResultError
	}
	e.emit(ev.EventName(), "", attrs, ev)
	ObserveAuctionCompleted(result)
}

func (e Event) DSPRequestSent(params Params, dsp string) {
	ev := dspRequestSent{
		attrs: attrsFrom(params),
		DSP:   dsp,
	}
	e.emit(ev.EventName(), dsp, ev.attrs, ev)
}

func (e Event) DSPResponseReceived(params Params, dr *adapters.DemandResponse) {
	if dr == nil {
		return
	}
	attrs := attrsFrom(params)
	latencyMS := dr.EndTS - dr.StartTS
	outcome := OutcomeFromDemand(dr.Error, dr.IsBid(), dr.Status)
	ev := dspResponseReceived{
		attrs:      attrs,
		DSP:        string(dr.DemandID),
		Outcome:    outcome,
		HTTPStatus: dr.Status,
		LatencyMS:  latencyMS,
	}
	if dr.IsBid() {
		ev.Price = dr.Price()
	}
	e.emit(ev.EventName(), ev.DSP, attrs, ev)
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
	ev := dspResponseRejected{
		attrs:      attrsFrom(params),
		DSP:        string(dr.DemandID),
		Price:      dr.Price(),
		PriceFloor: floor,
	}
	e.emit(ev.EventName(), ev.DSP, ev.attrs, ev)
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
