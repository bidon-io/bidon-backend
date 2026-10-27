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

// Identity is the join key for events that are not an auction request:
// impressions and notices. auction_id is the client-supplied id.
type Identity struct {
	AppID     int64
	AuctionID string
	SessionID string
	AdType    string
	AdFormat  string
	Country   string
}

func (id Identity) attrs() attrs {
	return attrs{
		AppID:     id.AppID,
		AuctionID: id.AuctionID,
		SessionID: id.SessionID,
		AdType:    id.AdType,
		AdFormat:  id.AdFormat,
		Country:   id.Country,
	}
}
