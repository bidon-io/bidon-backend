package telemetry

import (
	"context"
	"errors"
)

const (
	OutcomeBid       = "bid"
	OutcomeNoBid     = "nobid"
	OutcomeTimeout   = "timeout"
	OutcomeHTTPError = "http_error"
	OutcomeMalformed = "malformed"
)

const RejectReasonBelowFloor = "below_floor"

const ScopeBiddingRound = "bidding_round"

const (
	AuctionResultOK    = "ok"
	AuctionResultError = "error"
)

// OutcomeFromDemand maps a demand response to a catalog outcome.
// Precedence: timeout → http_error (4xx/5xx or status 0 with err) → malformed → bid → nobid.
func OutcomeFromDemand(err error, isBid bool, status int) string {
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
