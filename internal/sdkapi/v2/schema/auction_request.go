package schema

import (
	"github.com/bidon-io/bidon-backend/internal/ad"
)

// AuctionRequest Has both the auction and bidding request fields
type AuctionRequest struct {
	BaseRequest
	AdType   ad.Type  `param:"ad_type"`
	Adapters Adapters `json:"adapters" validate:"required"`
	AdObject AdObject `json:"imp" validate:"required"`
	Test     bool     `json:"test"` // Flag indicating that request is test
	TMax     int64    `json:"tmax"` // Max response time for server before timeout
}

func (r *AuctionRequest) GetAuctionConfigurationParams() (string, string) {
	return "", r.AdObject.AuctionConfigurationUID
}

func (r *AuctionRequest) SetAuctionConfigurationParams(id int64, uid string) {
	r.AdObject.AuctionConfigurationUID = uid
}

func (r *AuctionRequest) ToAuctionRequest() AuctionRequest {
	return AuctionRequest{
		BaseRequest: r.BaseRequest,
		AdType:      r.AdType,
		Adapters:    r.Adapters,
		AdObject: AdObject{
			AuctionID:               r.AdObject.AuctionID,
			AuctionConfigurationUID: r.AdObject.AuctionConfigurationUID,
			PriceFloor:              r.AdObject.PriceFloor,
			Banner:                  r.AdObject.Banner,
			Interstitial:            r.AdObject.Interstitial,
			Rewarded:                r.AdObject.Rewarded,
		},
	}
}

func (r *AuctionRequest) ToBiddingRequest(roundID string) BiddingRequest {
	return BiddingRequest{
		BaseRequest: r.BaseRequest,
		AdType:      r.AdType,
		Adapters:    r.Adapters,
		Imp:         r.AdObject.ToImp(roundID),
		Test:        r.Test,
		TMax:        r.TMax,
	}
}
