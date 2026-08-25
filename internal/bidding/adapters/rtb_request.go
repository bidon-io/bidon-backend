package adapters

import (
	"fmt"

	"github.com/gofrs/uuid/v5"
	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

// CustomRequestBuilder is implemented by non-OpenRTB adapters that fully own
// request construction. Type-asserted by BuildDemandRequest. Amazon uses
// FetchBids and never reaches this path.
type CustomRequestBuilder interface {
	CreateRequest(openrtb.BidRequest, *schema.AuctionRequest) (openrtb.BidRequest, error)
}

// OpenRTBRequestEnricher is implemented by OpenRTB adapters that need to
// mutate the request after the shared shell (token placement, ext, user).
type OpenRTBRequestEnricher interface {
	EnrichOpenRTBRequest(*openrtb.BidRequest, *schema.AuctionRequest) error
}

// RTBRequestOptions configures optional CreateRequest fields applied by BuildRTBRequest.
// Network-specific token placement and ext shapes stay in adapters (enrich after the build).
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

// BuildDemandRequest constructs the outbound OpenRTB request at the builder
// call site. Custom builders take precedence; otherwise the shared shell
// runs around BidderInterface.BuildImpression, optionally followed by an enricher.
func BuildDemandRequest(
	bidder BidderInterface,
	request openrtb.BidRequest,
	auctionRequest *schema.AuctionRequest,
	demandKey adapter.Key,
) (openrtb.BidRequest, error) {
	if c, ok := bidder.(CustomRequestBuilder); ok {
		return c.CreateRequest(request, auctionRequest)
	}

	imp, opts, err := bidder.BuildImpression(request, auctionRequest)
	if err != nil {
		return request, err
	}
	if imp == nil {
		return request, fmt.Errorf("nil impression")
	}

	request = BuildRTBRequest(request, auctionRequest, demandKey, *imp, opts)

	if e, ok := bidder.(OpenRTBRequestEnricher); ok {
		if err := e.EnrichOpenRTBRequest(&request, auctionRequest); err != nil {
			return request, err
		}
	}

	return request, nil
}

func CalculatePriceFloor(rtbRequest *openrtb.BidRequest, incomingRequest *schema.AuctionRequest) float64 {
	if rtbRequest == nil || incomingRequest == nil {
		return 0
	}

	if len(rtbRequest.Imp) == 1 {
		return rtbRequest.Imp[0].BidFloor
	}

	return incomingRequest.AdObject.GetBidFloorForBidding()
}

// BuildRTBRequest applies the common OpenRTB impression shell around an
// adapter-built Imp. Imp is required; callers must not pass a zero-value
// placeholder in place of a real creative. Size maps and banner / interstitial
// / rewarded Imp builders live in impression.go.
func BuildRTBRequest(
	request openrtb.BidRequest,
	auctionRequest *schema.AuctionRequest,
	demandKey adapter.Key,
	imp openrtb2.Imp,
	opts RTBRequestOptions,
) openrtb.BidRequest {
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

	request.Imp = []openrtb2.Imp{imp}
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
