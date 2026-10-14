package telemetry

const SchemaVersion = "0.1"

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
