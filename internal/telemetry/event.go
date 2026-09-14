package telemetry

import (
	"time"

	"github.com/gofrs/uuid/v5"

	"github.com/bidon-io/bidon-backend/config"
)

type Event interface {
	Topic() config.Topic
}

type Record struct {
	Envelope
	Scope            string  `json:"scope,omitempty"`
	DSP              string  `json:"dsp,omitempty"`
	Outcome          string  `json:"outcome,omitempty"`
	HTTPStatus       int     `json:"http_status,omitempty"`
	LatencyMS        int64   `json:"latency_ms,omitempty"`
	Price            float64 `json:"price,omitempty"`
	PriceFloor       float64 `json:"price_floor,omitempty"`
	WinnerDSP        string  `json:"winner_dsp,omitempty"`
	ParticipantCount int     `json:"participant_count,omitempty"`
	TotalLatencyMS   int64   `json:"total_latency_ms,omitempty"`
	ErrorCode        string  `json:"error_code,omitempty"`
	RejectReason     string  `json:"reject_reason,omitempty"`
}

func (r Record) Topic() config.Topic { return config.TelemetryEventsTopic }

func NewRecord(name string, env Envelope) Record {
	if env.EventID == "" {
		id, _ := uuid.NewV4()
		env.EventID = id.String()
	}
	if env.EventTS == 0 {
		env.EventTS = time.Now().UnixMilli()
	}
	env.EventName = name
	env.SchemaVersion = SchemaVersion
	env.SamplingRate = 1.0

	return Record{Envelope: env}
}
