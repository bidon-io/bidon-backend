package schema

import (
	"github.com/bidon-io/bidon-backend/internal/ad"
)

// HybridAuctionRequest Has both the auction and bidding request fields
type HybridAuctionRequest struct {
	BaseRequest
	AdType         ad.Type        `param:"ad_type"`
	Adapters       Adapters       `json:"adapters" validate:"required"`
	HybridAdObject HybridAdObject `json:"imp" validate:"required"`
	Test           bool           `json:"test"` // Flag indicating that request is test
	TMax           int64          `json:"tmax"` // Max response time for server before timeout
}

func (r *HybridAuctionRequest) GetAuctionConfigurationParams() (string, string) {
	return "", r.HybridAdObject.AuctionConfigurationUID
}

func (r *HybridAuctionRequest) SetAuctionConfigurationParams(id int64, uid string) {
	r.HybridAdObject.AuctionConfigurationUID = uid
}

func (r *HybridAuctionRequest) ToAuctionRequest() AuctionRequest {
	return AuctionRequest{
		BaseRequest: r.BaseRequest,
		AdType:      r.AdType,
		Adapters:    r.Adapters,
		AdObject: AdObject{
			AuctionID:               r.HybridAdObject.AuctionID,
			AuctionConfigurationUID: r.HybridAdObject.AuctionConfigurationUID,
			PriceFloor:              r.HybridAdObject.PriceFloor,
			Banner:                  r.HybridAdObject.Banner,
			Interstitial:            r.HybridAdObject.Interstitial,
			Rewarded:                r.HybridAdObject.Rewarded,
		},
	}
}

func (r *HybridAuctionRequest) ToBiddingRequest(roundID string) BiddingRequest {
	return BiddingRequest{
		BaseRequest: r.BaseRequest,
		AdType:      r.AdType,
		Adapters:    r.Adapters,
		Imp:         r.HybridAdObject.ToImp(roundID),
		Test:        r.Test,
		TMax:        r.TMax,
	}
}
