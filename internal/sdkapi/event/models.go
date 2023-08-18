package event

import (
	"time"

	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

type BidRequest struct {
	RawRequest                  *schema.BiddingRequest `json:"raw_request"`
	Timestamp                   time.Time              `json:"timestamp"`
	EventType                   string                 `json:"event_type"`
	AdType                      string                 `json:"ad_type"`
	AuctionID                   string                 `json:"auction_id"`
	AuctionConfigurationID      int64                  `json:"auction_configuration_id"`
	AuctionStatus               string                 `json:"auction_status"`
	RoundID                     string                 `json:"round_id"`
	RoundNumber                 int                    `json:"round_number"`
	ImpID                       string                 `json:"impid"`
	DemandID                    string                 `json:"demand_id"`
	AdUnitID                    int                    `json:"ad_unit_id"`
	AdUnitCode                  string                 `json:"ad_unit_code"`
	Ecpm                        float64                `json:"ecpm"`
	PriceFloor                  float64                `json:"price_floor"`
	Manufacturer                string                 `json:"manufacturer"`
	Model                       string                 `json:"model"`
	Os                          string                 `json:"os"`
	OsVersion                   string                 `json:"os_version"`
	ConnectionType              string                 `json:"connection_type"`
	SessionID                   string                 `json:"session_id"`
	SessionUptime               int                    `json:"session_uptime"`
	Bundle                      string                 `json:"bundle"`
	Framework                   string                 `json:"framework"`
	FrameworkVersion            string                 `json:"framework_version"`
	PluginVersion               string                 `json:"plugin_version"`
	PackageVersion              string                 `json:"package_version"`
	SdkVersion                  string                 `json:"sdk_version"`
	IDFA                        string                 `json:"idfa"`
	IDG                         string                 `json:"idg"`
	IDFV                        string                 `json:"idfv"`
	TrackingAuthorizationStatus string                 `json:"tracking_authorization_status"`
	COPPA                       bool                   `json:"coppa"`
	GDPR                        bool                   `json:"gdpr"`
	CountryCode                 string                 `json:"country_code"`
	City                        string                 `json:"city"`
	Ip                          string                 `json:"ip"`
	CountryID                   int64                  `json:"country_id"`
	SegmentID                   int64                  `json:"segment_id"`
	Ext                         string                 `json:"ext"`
}
