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
	// Base is the auction or show request. Nil when the caller has no device
	// or consent payload to copy.
	Base *schema.BaseRequest
}

// Identity is the join key for events that are not an auction request:
// impressions and notices. auction_id is the client-supplied id.
type Identity struct {
	AppID     int64
	AuctionID string
	SessionID string
	AdType    string
	AdFormat  string
	Country   string
	Base      *schema.BaseRequest
}

func (id Identity) attrs() attrs {
	return attrs{
		AppID:     id.AppID,
		AuctionID: id.AuctionID,
		SessionID: id.SessionID,
		AdType:    id.AdType,
		AdFormat:  id.AdFormat,
		Country:   id.Country,
		Base:      id.Base,
	}
}
