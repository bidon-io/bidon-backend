package schema

import (
	"github.com/bidon-io/bidon-backend/internal/ad"
)

// HybridAuctionRequest Has both the auction and bidding request fields
type HybridAuctionRequest struct {
	BaseRequest
	AdType    ad.Type   `param:"ad_type"`
	Adapters  Adapters  `json:"adapters" validate:"required"`
	HybridImp HybridImp `json:"imp" validate:"required"`
	Test      bool      `json:"test"` // Flag indicating that request is test
	TMax      int64     `json:"tmax"` // Max response time for server before timeout
}

func (r *HybridAuctionRequest) GetAuctionConfigurationParams() (string, string) {
	return "", r.HybridImp.AuctionConfigurationUID
}

func (r *HybridAuctionRequest) SetAuctionConfigurationParams(id int64, uid string) {
	r.HybridImp.AuctionConfigurationUID = uid
}

func (r *HybridAuctionRequest) ToAuctionRequest() AuctionRequest {
	return AuctionRequest{
		BaseRequest: r.BaseRequest,
		AdType:      r.AdType,
		Adapters:    r.Adapters,
		AdObject: AdObject{
			AuctionID:               r.HybridImp.AuctionID,
			AuctionConfigurationUID: r.HybridImp.AuctionConfigurationUID,
			PriceFloor:              r.HybridImp.PriceFloor,
			Banner:                  r.HybridImp.Banner,
			Interstitial:            r.HybridImp.Interstitial,
			Rewarded:                r.HybridImp.Rewarded,
		},
	}
}

func (r *HybridAuctionRequest) ToBiddingRequest() BiddingRequest {
	return BiddingRequest{
		BaseRequest: r.BaseRequest,
		AdType:      r.AdType,
		Adapters:    r.Adapters,
		Imp:         r.HybridImp.ToImp(),
		Test:        r.Test,
		TMax:        r.TMax,
	}
}
