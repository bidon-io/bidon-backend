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
