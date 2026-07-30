package adapters

import (
	"github.com/gofrs/uuid/v5"
	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

// RTBRequestOptions configures optional CreateRequest fields applied by BuildRTBRequest.
// Network-specific token placement and ext shapes stay in adapters (mutate after the build).
type RTBRequestOptions struct {
	TagID       string
	AppID       string
	PublisherID string
	BuyerUID    string

	// OmitBidFloorCur leaves BidFloorCur unset. Default is "USD".
	OmitBidFloorCur bool
	// OmitSecure leaves Secure unset. Default is 1.
	OmitSecure bool
	// OmitDisplayManager leaves DisplayManager and DisplayManagerVer unset.
	OmitDisplayManager bool
}

// BuildRTBRequest applies the common OpenRTB CreateRequest impression shell around
// an adapter-built creative Imp. Creative construction (banner/interstitial/rewarded)
// remains adapter-owned until BAC-28.
func BuildRTBRequest(
	request openrtb.BidRequest,
	auctionRequest *schema.AuctionRequest,
	demandKey adapter.Key,
	imp *openrtb2.Imp,
	opts RTBRequestOptions,
) openrtb.BidRequest {
	if imp == nil {
		return request
	}

	impID, _ := uuid.NewV4()
	imp.ID = impID.String()

	if opts.TagID != "" {
		imp.TagID = opts.TagID
	}

	if !opts.OmitDisplayManager {
		imp.DisplayManager = string(demandKey)
		if auctionRequest != nil {
			if info, ok := auctionRequest.Adapters[demandKey]; ok {
				imp.DisplayManagerVer = info.SDKVersion
			}
		}
	}

	if !opts.OmitSecure {
		secure := int8(1)
		imp.Secure = &secure
	}

	imp.BidFloor = CalculatePriceFloor(&request, auctionRequest)
	if !opts.OmitBidFloorCur {
		imp.BidFloorCur = "USD"
	}

	request.Imp = []openrtb2.Imp{*imp}
	request.Cur = []string{"USD"}

	if opts.AppID != "" || opts.PublisherID != "" {
		if request.App == nil {
			request.App = &openrtb2.App{}
		}
		if opts.AppID != "" {
			request.App.ID = opts.AppID
		}
		if opts.PublisherID != "" {
			if request.App.Publisher == nil {
				request.App.Publisher = &openrtb2.Publisher{}
			}
			request.App.Publisher.ID = opts.PublisherID
		}
	}

	if opts.BuyerUID != "" {
		if request.User == nil {
			request.User = &openrtb.User{}
		}
		request.User.BuyerUID = opts.BuyerUID
	}

	return request
}
