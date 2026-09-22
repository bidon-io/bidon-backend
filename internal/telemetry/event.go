package telemetry

import (
	"time"

	"github.com/gofrs/uuid/v5"
	"google.golang.org/protobuf/proto"

	telemetryv1 "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/telemetry/v1"
)

func newProtoEnvelope(a attrs, name string) *telemetryv1.Envelope {
	id, _ := uuid.NewV4()
	return &telemetryv1.Envelope{
		EventId:       id.String(),
		EventName:     name,
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

func outcomeFromProto(o telemetryv1.Outcome) Outcome {
	switch o {
	case telemetryv1.Outcome_OUTCOME_BID:
		return OutcomeBid
	case telemetryv1.Outcome_OUTCOME_NOBID:
		return OutcomeNoBid
	case telemetryv1.Outcome_OUTCOME_TIMEOUT:
		return OutcomeTimeout
	case telemetryv1.Outcome_OUTCOME_HTTP_ERROR:
		return OutcomeHTTPError
	case telemetryv1.Outcome_OUTCOME_MALFORMED:
		return OutcomeMalformed
	default:
		return ""
	}
}

func scopeFromProto(s telemetryv1.Scope) Scope {
	if s == telemetryv1.Scope_SCOPE_BIDDING_ROUND {
		return ScopeBiddingRound
	}
	return ""
}

func errorCodeFromProto(c telemetryv1.ErrorCode) ErrorCode {
	switch c {
	case telemetryv1.ErrorCode_ERROR_CODE_NO_ADS_FOUND:
		return ErrorCodeNoAdsFound
	case telemetryv1.ErrorCode_ERROR_CODE_INVALID_AUCTION_KEY:
		return ErrorCodeInvalidAuctionKey
	case telemetryv1.ErrorCode_ERROR_CODE_ERROR:
		return ErrorCodeError
	default:
		return ""
	}
}

func rejectReasonFromProto(r telemetryv1.RejectReason) RejectReason {
	if r == telemetryv1.RejectReason_REJECT_REASON_BELOW_FLOOR {
		return RejectReasonBelowFloor
	}
	return ""
}

func recordFromEnvelope(env *telemetryv1.Envelope) Record {
	if env == nil {
		return Record{}
	}
	return Record{
		Envelope: Envelope{
			EventID:       env.GetEventId(),
			EventName:     env.GetEventName(),
			EventTS:       env.GetEventTs(),
			SchemaVersion: env.GetSchemaVersion(),
			AppID:         env.GetAppId(),
			AuctionID:     env.GetAuctionId(),
			SessionID:     env.GetSessionId(),
			AdType:        env.GetAdType(),
			AdFormat:      env.GetAdFormat(),
			Country:       env.GetCountry(),
			TraceID:       env.GetTraceId(),
			SamplingRate:  env.GetSamplingRate(),
		},
	}
}

func recordFromProto(msg proto.Message) Record {
	switch m := msg.(type) {
	case *telemetryv1.AuctionRequestReceived:
		rec := recordFromEnvelope(m.GetEnvelope())
		rec.PriceFloor = m.GetPriceFloor()
		return rec
	case *telemetryv1.AuctionCompleted:
		rec := recordFromEnvelope(m.GetEnvelope())
		rec.Scope = scopeFromProto(m.GetScope())
		rec.WinnerDSP = m.GetWinnerDsp()
		rec.Price = m.GetPrice()
		rec.ParticipantCount = int(m.GetParticipantCount())
		rec.TotalLatencyMS = m.GetTotalLatencyMs()
		rec.ErrorCode = errorCodeFromProto(m.GetErrorCode())
		return rec
	case *telemetryv1.DspRequestSent:
		rec := recordFromEnvelope(m.GetEnvelope())
		rec.Scope = scopeFromProto(m.GetScope())
		rec.DSP = m.GetDsp()
		return rec
	case *telemetryv1.DspResponseReceived:
		rec := recordFromEnvelope(m.GetEnvelope())
		rec.Scope = scopeFromProto(m.GetScope())
		rec.DSP = m.GetDsp()
		rec.Outcome = outcomeFromProto(m.GetOutcome())
		rec.HTTPStatus = int(m.GetHttpStatus())
		rec.LatencyMS = m.GetLatencyMs()
		rec.Price = m.GetPrice()
		return rec
	case *telemetryv1.DspResponseRejected:
		rec := recordFromEnvelope(m.GetEnvelope())
		rec.Scope = scopeFromProto(m.GetScope())
		rec.DSP = m.GetDsp()
		rec.Price = m.GetPrice()
		rec.PriceFloor = m.GetPriceFloor()
		rec.RejectReason = rejectReasonFromProto(m.GetRejectReason())
		return rec
	default:
		return Record{}
	}
}

// Record is the decoded catalog view used by tests and the log engine.
type Record struct {
	Envelope
	Scope            Scope
	DSP              string
	Outcome          Outcome
	HTTPStatus       int
	LatencyMS        int64
	Price            float64
	PriceFloor       float64
	WinnerDSP        string
	ParticipantCount int
	TotalLatencyMS   int64
	ErrorCode        ErrorCode
	RejectReason     RejectReason
}
