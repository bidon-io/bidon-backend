package telemetry

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	telemetryv1 "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/telemetry/v1"
)

func envelopeToProto(env Envelope) *telemetryv1.Envelope {
	return &telemetryv1.Envelope{
		EventId:       env.EventID,
		EventName:     env.EventName,
		EventTs:       env.EventTS,
		SchemaVersion: env.SchemaVersion,
		AppId:         env.AppID,
		AuctionId:     env.AuctionID,
		SessionId:     env.SessionID,
		AdType:        env.AdType,
		AdFormat:      env.AdFormat,
		Country:       env.Country,
		TraceId:       env.TraceID,
		SamplingRate:  env.SamplingRate,
	}
}

func recordToProto(rec Record) (proto.Message, error) {
	env := envelopeToProto(rec.Envelope)
	switch rec.EventName {
	case EventAuctionRequestReceived:
		return &telemetryv1.AuctionRequestReceived{
			Envelope:   env,
			PriceFloor: rec.PriceFloor,
		}, nil
	case EventAuctionCompleted:
		return &telemetryv1.AuctionCompleted{
			Envelope:         env,
			Scope:            protoScope(rec.Scope),
			WinnerDsp:        rec.WinnerDSP,
			Price:            rec.Price,
			ParticipantCount: int32(rec.ParticipantCount),
			TotalLatencyMs:   rec.TotalLatencyMS,
			ErrorCode:        protoErrorCode(rec.ErrorCode),
		}, nil
	case EventDSPRequestSent:
		return &telemetryv1.DspRequestSent{
			Envelope: env,
			Scope:    protoScope(rec.Scope),
			Dsp:      rec.DSP,
		}, nil
	case EventDSPResponseReceived:
		return &telemetryv1.DspResponseReceived{
			Envelope:   env,
			Scope:      protoScope(rec.Scope),
			Dsp:        rec.DSP,
			Outcome:    protoOutcome(rec.Outcome),
			HttpStatus: int32(rec.HTTPStatus),
			LatencyMs:  rec.LatencyMS,
			Price:      rec.Price,
		}, nil
	case EventDSPResponseRejected:
		return &telemetryv1.DspResponseRejected{
			Envelope:     env,
			Scope:        protoScope(rec.Scope),
			Dsp:          rec.DSP,
			Price:        rec.Price,
			PriceFloor:   rec.PriceFloor,
			RejectReason: protoRejectReason(rec.RejectReason),
		}, nil
	default:
		return nil, fmt.Errorf("unknown event_name %q", rec.EventName)
	}
}

func protoScope(s string) telemetryv1.Scope {
	if s == "bidding_round" {
		return telemetryv1.Scope_SCOPE_BIDDING_ROUND
	}
	return telemetryv1.Scope_SCOPE_UNSPECIFIED
}

func protoOutcome(s string) telemetryv1.Outcome {
	switch s {
	case "bid":
		return telemetryv1.Outcome_OUTCOME_BID
	case "nobid":
		return telemetryv1.Outcome_OUTCOME_NOBID
	case "timeout":
		return telemetryv1.Outcome_OUTCOME_TIMEOUT
	case "http_error":
		return telemetryv1.Outcome_OUTCOME_HTTP_ERROR
	case "malformed":
		return telemetryv1.Outcome_OUTCOME_MALFORMED
	default:
		return telemetryv1.Outcome_OUTCOME_UNSPECIFIED
	}
}

func protoErrorCode(s string) telemetryv1.ErrorCode {
	switch s {
	case "no_ads_found":
		return telemetryv1.ErrorCode_ERROR_CODE_NO_ADS_FOUND
	case "invalid_auction_key":
		return telemetryv1.ErrorCode_ERROR_CODE_INVALID_AUCTION_KEY
	case "error":
		return telemetryv1.ErrorCode_ERROR_CODE_ERROR
	default:
		return telemetryv1.ErrorCode_ERROR_CODE_UNSPECIFIED
	}
}

func protoRejectReason(s string) telemetryv1.RejectReason {
	if s == "below_floor" {
		return telemetryv1.RejectReason_REJECT_REASON_BELOW_FLOOR
	}
	return telemetryv1.RejectReason_REJECT_REASON_UNSPECIFIED
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

func scopeFromProto(s telemetryv1.Scope) string {
	if s == telemetryv1.Scope_SCOPE_BIDDING_ROUND {
		return "bidding_round"
	}
	return ""
}

func outcomeFromProto(o telemetryv1.Outcome) string {
	switch o {
	case telemetryv1.Outcome_OUTCOME_BID:
		return "bid"
	case telemetryv1.Outcome_OUTCOME_NOBID:
		return "nobid"
	case telemetryv1.Outcome_OUTCOME_TIMEOUT:
		return "timeout"
	case telemetryv1.Outcome_OUTCOME_HTTP_ERROR:
		return "http_error"
	case telemetryv1.Outcome_OUTCOME_MALFORMED:
		return "malformed"
	default:
		return ""
	}
}

func errorCodeFromProto(c telemetryv1.ErrorCode) string {
	switch c {
	case telemetryv1.ErrorCode_ERROR_CODE_NO_ADS_FOUND:
		return "no_ads_found"
	case telemetryv1.ErrorCode_ERROR_CODE_INVALID_AUCTION_KEY:
		return "invalid_auction_key"
	case telemetryv1.ErrorCode_ERROR_CODE_ERROR:
		return "error"
	default:
		return ""
	}
}

func rejectReasonFromProto(r telemetryv1.RejectReason) string {
	if r == telemetryv1.RejectReason_REJECT_REASON_BELOW_FLOOR {
		return "below_floor"
	}
	return ""
}
