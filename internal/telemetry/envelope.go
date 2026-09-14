package telemetry

const SchemaVersion = "0.1"

const (
	EventAuctionRequestReceived = "auction_request_received"
	EventAuctionCompleted       = "auction_completed"
	EventDSPRequestSent         = "dsp_request_sent"
	EventDSPResponseReceived    = "dsp_response_received"
	EventDSPResponseRejected    = "dsp_response_rejected"
)

type Envelope struct {
	EventID       string  `json:"event_id"`
	EventName     string  `json:"event_name"`
	EventTS       int64   `json:"event_ts"`
	SchemaVersion string  `json:"schema_version"`
	AppID         int64   `json:"app_id"`
	AuctionID     string  `json:"auction_id"`
	SessionID     string  `json:"session_id"`
	AdType        string  `json:"ad_type,omitempty"`
	AdFormat      string  `json:"ad_format,omitempty"`
	Country       string  `json:"country,omitempty"`
	TraceID       string  `json:"trace_id,omitempty"`
	SamplingRate  float64 `json:"sampling_rate"`
}
