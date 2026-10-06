package telemetry

import (
	"time"

	"github.com/gofrs/uuid/v5"

	telemetryv1 "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/telemetry/v1"
)

// newProtoEnvelope builds the shared header. emit fills event_name from the
// message type's (event_name) option.
func newProtoEnvelope(a attrs) *telemetryv1.Envelope {
	id, _ := uuid.NewV4()
	return &telemetryv1.Envelope{
		EventId:       id.String(),
		EventTs:       time.Now().UnixMilli(),
		SchemaVersion: SchemaVersion,
		AppId:         a.AppID,
		AuctionId:     a.AuctionID,
		SessionId:     a.SessionID,
		AdType:        a.AdType,
		AdFormat:      a.AdFormat,
		Country:       a.Country,
		TraceId:       a.TraceID,
		SamplingRate:  1.0,
	}
}

func protoOutcome(o Outcome) telemetryv1.Outcome {
	switch o {
	case OutcomeBid:
		return telemetryv1.Outcome_OUTCOME_BID
	case OutcomeNoBid:
		return telemetryv1.Outcome_OUTCOME_NOBID
	case OutcomeTimeout:
		return telemetryv1.Outcome_OUTCOME_TIMEOUT
	case OutcomeHTTPError:
		return telemetryv1.Outcome_OUTCOME_HTTP_ERROR
	case OutcomeMalformed:
		return telemetryv1.Outcome_OUTCOME_MALFORMED
	default:
		return telemetryv1.Outcome_OUTCOME_UNSPECIFIED
	}
}

func protoErrorCode(c ErrorCode) telemetryv1.ErrorCode {
	switch c {
	case ErrorCodeNoAdsFound:
		return telemetryv1.ErrorCode_ERROR_CODE_NO_ADS_FOUND
	case ErrorCodeInvalidAuctionKey:
		return telemetryv1.ErrorCode_ERROR_CODE_INVALID_AUCTION_KEY
	case ErrorCodeError:
		return telemetryv1.ErrorCode_ERROR_CODE_ERROR
	default:
		return telemetryv1.ErrorCode_ERROR_CODE_UNSPECIFIED
	}
}
