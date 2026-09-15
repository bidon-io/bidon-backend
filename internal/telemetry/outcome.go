package telemetry

import (
	"context"
	"errors"
)

type Outcome string

const (
	OutcomeBid       Outcome = "bid"
	OutcomeNoBid     Outcome = "nobid"
	OutcomeTimeout   Outcome = "timeout"
	OutcomeHTTPError Outcome = "http_error"
	OutcomeMalformed Outcome = "malformed"
)

type Scope string

const ScopeBiddingRound Scope = "bidding_round"

type RejectReason string

const RejectReasonBelowFloor RejectReason = "below_floor"

type AuctionResult string

const (
	AuctionResultOK    AuctionResult = "ok"
	AuctionResultError AuctionResult = "error"
)

type ErrorCode string

const (
	ErrorCodeNoAdsFound        ErrorCode = "no_ads_found"
	ErrorCodeInvalidAuctionKey ErrorCode = "invalid_auction_key"
	ErrorCodeError             ErrorCode = "error"
)

// OutcomeFromDemand maps a demand response to a catalog outcome.
// Precedence: timeout → http_error (4xx/5xx or status 0 with err) → malformed → bid → nobid.
func OutcomeFromDemand(err error, isBid bool, status int) Outcome {
	if errors.Is(err, context.DeadlineExceeded) {
		return OutcomeTimeout
	}
	if status >= 400 {
		return OutcomeHTTPError
	}
	if err != nil && status == 0 {
		return OutcomeHTTPError
	}
	if err != nil {
		return OutcomeMalformed
	}
	if isBid {
		return OutcomeBid
	}
	return OutcomeNoBid
}
