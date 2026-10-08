package telemetry

import (
	"context"
	"errors"
	"strings"

	telemetryv1 "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/telemetry/v1"
)

type AuctionResult string

const (
	AuctionResultOK    AuctionResult = "ok"
	AuctionResultError AuctionResult = "error"
)

// OutcomeFromDemand maps a demand response to a catalog outcome.
// Precedence: timeout → http_error (4xx/5xx or status 0 with err) → malformed → bid → nobid.
func OutcomeFromDemand(err error, isBid bool, status int) telemetryv1.Outcome {
	if errors.Is(err, context.DeadlineExceeded) {
		return telemetryv1.Outcome_OUTCOME_TIMEOUT
	}
	if status >= 400 {
		return telemetryv1.Outcome_OUTCOME_HTTP_ERROR
	}
	if err != nil && status == 0 {
		return telemetryv1.Outcome_OUTCOME_HTTP_ERROR
	}
	if err != nil {
		return telemetryv1.Outcome_OUTCOME_MALFORMED
	}
	if isBid {
		return telemetryv1.Outcome_OUTCOME_BID
	}
	return telemetryv1.Outcome_OUTCOME_NOBID
}

// outcomeLabel is the dsp_response_total outcome label: the enum value name
// without its prefix, lower-cased (OUTCOME_HTTP_ERROR → http_error).
func outcomeLabel(o telemetryv1.Outcome) string {
	return strings.ToLower(strings.TrimPrefix(o.String(), "OUTCOME_"))
}
