package telemetry

import "github.com/bidon-io/bidon-backend/internal/sdkapi/schema"

// SchemaVersion is the current telemetry major. Additive envelope and catalog
// changes stay on this number.
const SchemaVersion int32 = 1

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
	// Base is the auction request. Nil when the caller has no device or
	// consent payload to copy.
	Base *schema.BaseRequest
}
