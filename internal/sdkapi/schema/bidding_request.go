package schema

import (
	"strconv"
	"strings"

	"github.com/bidon-io/bidon-backend/internal/ad"
)

type BiddingRequest struct {
	BaseRequest
	AdType   ad.Type  `param:"ad_type"`
	Adapters Adapters `json:"adapters" validate:"required"`
	Imp      Imp      `json:"imp" validate:"required"`
	Test     bool     `json:"test"` // Flag indicating that request is test
	TMax     int      `json:"tmax"` // Max response time for server before timeout
}

func (b *BiddingRequest) NormalizeValues() {
	b.BaseRequest.NormalizeValues()
	b.Imp.ID = strings.ToLower(b.Imp.ID)
	b.Imp.AuctionID = strings.ToLower(b.Imp.AuctionID)
}

func (b *BiddingRequest) GetAuctionConfigurationParams(sdkVersion string) (string, string) {
	if b.Imp.AuctionConfigurationUID != "" {
		return "uid", b.Imp.AuctionConfigurationUID
	}

	return "id", strconv.FormatInt(b.Imp.AuctionConfigurationID, 10)
}

func (b *BiddingRequest) SetAuctionConfigurationParams(id int64, uid string) {
	b.Imp.AuctionConfigurationID = id
	b.Imp.AuctionConfigurationUID = uid
}
