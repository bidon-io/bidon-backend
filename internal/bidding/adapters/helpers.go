package adapters

import (
	"fmt"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

func CalculatePriceFloor(rtbRequest *openrtb.BidRequest, incomingRequest *schema.AuctionRequest) float64 {
	if rtbRequest == nil || incomingRequest == nil {
		return 0
	}

	if len(rtbRequest.Imp) == 1 {
		return rtbRequest.Imp[0].BidFloor
	} else {
		return incomingRequest.AdObject.GetBidFloorForBidding()
	}
}

var FullscreenFormats = map[string][2]int64{
	"PHONE":  {320, 480},
	"TABLET": {768, 1024},
}

// BannerFormats maps common banner ad formats to OpenRTB WxH.
// Adaptive tablet sizing is handled by ResolveBannerSize (leaderboard upgrade).
var BannerFormats = map[ad.Format][2]int64{
	ad.BannerFormat:      {320, 50},
	ad.LeaderboardFormat: {728, 90},
	ad.MRECFormat:        {300, 250},
	ad.AdaptiveFormat:    {320, 50},
	ad.EmptyFormat:       {320, 50}, // Default
}

// BannerSizeOptions configures ResolveBannerSize behavior for networks with
// non-default sizing rules.
type BannerSizeOptions struct {
	// RejectAdaptiveLeaderboard returns an error when adaptive format on tablet
	// would resolve to leaderboard (e.g. bigoads does not support leaderboard).
	RejectAdaptiveLeaderboard bool
}

// ResolveBannerSize returns WxH for the given banner format.
// Unknown formats fall back to EmptyFormat (320×50).
// Adaptive + tablet maps to LeaderboardFormat unless RejectAdaptiveLeaderboard is set.
func ResolveBannerSize(format ad.Format, isAdaptive, isTablet bool, opts BannerSizeOptions) ([2]int64, error) {
	if isAdaptive && isTablet {
		if opts.RejectAdaptiveLeaderboard {
			return [2]int64{}, fmt.Errorf("unknown banner format: %s", format)
		}
		return BannerFormats[ad.LeaderboardFormat], nil
	}

	size, ok := BannerFormats[format]
	if !ok {
		size = BannerFormats[ad.EmptyFormat]
	}
	return size, nil
}
