package insights

import (
	"context"

	insightsopenrtb "github.com/bidon-io/bidon-backend/internal/insights/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/geocoder"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

type Key string

const (
	NeftaKey Key = "nefta"
)

type Service interface {
	Register(provider Provider) error
	Init(ctx context.Context, req InitRequest)
	FloorPrice(ctx context.Context, req FloorPriceRequest) []FloorPriceResult
}

type Provider interface {
	Key() Key
	Init(ctx context.Context, req InitRequest) (InitResult, error)
}

// FloorPriceProvider is an optional provider capability.
// Providers that only support Init should implement Provider only.
type FloorPriceProvider interface {
	Provider
	FloorPrice(ctx context.Context, req FloorPriceRequest) (FloorPriceResult, error)
}

// InitRequest is a provider-agnostic payload used by Insights providers.
type InitRequest struct {
	AppID       int64
	NUID        string
	SessionID   int64
	BaseRequest *schema.BaseRequest
	GeoData     geocoder.GeoData
	IDFA        string
	IDG         string
	IDFV        string
	AppVersion  string
	SDKVersion  string
	OpenRTB     insightsopenrtb.InitRequest
	Settings    map[string]any
}

type InitResult struct {
	Provider          Key
	RawRequest        string
	RawRequestHeaders string
	RawResponse       string
	Status            int
	Error             string
	Skipped           bool
}

// FloorPriceRequest is a provider-agnostic payload used by Insights providers
// that support floor-price recommendations.
type FloorPriceRequest struct {
	AppID                   int64
	NUID                    string
	SessionID               int64
	AdOpportunityID         int64
	SessionStartTS          int64
	AuctionID               string
	AuctionConfigurationID  int64
	AuctionConfigurationUID int64
	AdType                  string
	AdFormat                string
	BaseRequest             *schema.BaseRequest
	GeoData                 geocoder.GeoData
	IDFA                    string
	IDG                     string
	IDFV                    string
	AppVersion              string
	SDKVersion              string
	OpenRTB                 insightsopenrtb.InitRequest
	FloorPrice              float64
	Bidders                 []string
	Settings                map[string]any
}

type FloorPriceResult struct {
	Provider          Key
	Control           *bool
	Auction           *FloorPriceRecommendation
	RawRequest        string
	RawRequestHeaders string
	RawResponse       string
	Status            int
	Error             string
	Skipped           bool
}

type FloorPriceRecommendation struct {
	AuctionID    int32
	FloorPrice   float64
	Accuracy     any
	Notification FloorPriceNotification
}

type FloorPriceNotification struct {
	Auction    string
	Impression string
	Click      string
}
