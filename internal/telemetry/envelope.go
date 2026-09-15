package telemetry

const SchemaVersion = "0.1"

const (
	EventAuctionRequestReceived = "auction_request_received"
	EventAuctionCompleted       = "auction_completed"
	EventDSPRequestSent         = "dsp_request_sent"
	EventDSPResponseReceived    = "dsp_response_received"
	EventDSPResponseRejected    = "dsp_response_rejected"
)

// attrs is the identity every catalog event carries. Callers do not set this;
// Event methods stamp it from Params.
type attrs struct {
	AppID     int64
	AuctionID string
	SessionID string
	AdType    string
	AdFormat  string
	Country   string
	TraceID   string
}

// Envelope is the on-wire header. It is not part of the emit API.
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
