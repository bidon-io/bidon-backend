package startio

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/prebid/openrtb/v19/adcom1"
	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters/geo"
	"github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

// Adapter represents the Start.io bidding adapter.
type Adapter struct {
	TagID   string
	AppID   string
	Account string
}

var _ adapters.BidderInterface = (*Adapter)(nil)

// banner creates a banner impression for the bid request.
func (a *Adapter) banner(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	imp, _ := adapters.BuildBannerImp(auctionRequest, adapters.BannerImpOptions{})
	return imp
}

// interstitial creates an interstitial impression for the bid request.
func (a *Adapter) interstitial(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	return adapters.BuildInterstitialImp(auctionRequest, adapters.InterstitialImpOptions{
		IncludeVideo: true,
	})
}

// rewarded creates a rewarded video impression for the bid request.
func (a *Adapter) rewarded(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	skip := int8(1)
	return adapters.BuildRewardedImp(auctionRequest, adapters.RewardedImpOptions{
		IncludeBanner: true,
		Rwdd:          1,
		Skip:          &skip,
		BannerBAttr:   []adcom1.CreativeAttribute{16},
		VideoBAttr:    []adcom1.CreativeAttribute{1, 2, 5, 8, 9, 14, 17},
	})
}

func (a *Adapter) BuildImpression(_ openrtb.BidRequest, auctionRequest *schema.AuctionRequest) (*openrtb2.Imp, adapters.RTBRequestOptions, error) {
	if a.TagID == "" {
		return nil, adapters.RTBRequestOptions{}, errors.New("startio tag ID is empty")
	}

	if a.Account == "" {
		return nil, adapters.RTBRequestOptions{}, errors.New("startio account is empty")
	}

	if a.AppID == "" {
		return nil, adapters.RTBRequestOptions{}, errors.New("startio app ID is empty")
	}

	demandData, ok := auctionRequest.AdObject.Demands[adapter.StartIOKey]
	if !ok {
		return nil, adapters.RTBRequestOptions{}, errors.New("startio demand data missing")
	}

	token, ok := demandData["token"].(string)
	if !ok || token == "" {
		return nil, adapters.RTBRequestOptions{}, errors.New("startio token is empty")
	}

	var imp *openrtb2.Imp
	switch auctionRequest.AdObject.Type() {
	case ad.BannerType:
		imp = a.banner(auctionRequest)
	case ad.InterstitialType:
		imp = a.interstitial(auctionRequest)
	case ad.RewardedType:
		imp = a.rewarded(auctionRequest)
	default:
		return nil, adapters.RTBRequestOptions{}, errors.New("unknown impression type")
	}

	return imp, adapters.RTBRequestOptions{
		TagID:    a.TagID,
		AppID:    a.AppID,
		BuyerUID: token,
	}, nil
}

func (a *Adapter) EnrichOpenRTBRequest(request *openrtb.BidRequest, auctionRequest *schema.AuctionRequest) error {
	if auctionRequest.Test {
		request.Test = 1
	}

	// Start.io expects an empty publisher object even when no publisher id is set.
	if request.App == nil {
		request.App = &openrtb2.App{}
	}
	request.App.Publisher = &openrtb2.Publisher{}

	return nil
}

// ExecuteRequest implements the BidderInterface.ExecuteRequest method.
func (a *Adapter) ExecuteRequest(ctx context.Context, client *http.Client, request openrtb.BidRequest) *adapters.DemandResponse {
	endpoint := endpointByRegion(adapters.CountryFromRequest(request))
	if endpoint == "" {
		return &adapters.DemandResponse{
			DemandID:  adapter.StartIOKey,
			RequestID: request.ID,
			TagID:     a.TagID,
			Error:     errors.New("startio endpoint is empty"),
		}
	}

	return adapters.ExecuteRTBRequest(ctx, client, request, adapters.ExecuteRTBOptions{
		DemandID: adapter.StartIOKey,
		URL:      endpoint,
		TagID:    a.TagID,
		PrepareURL: func(base string, req openrtb.BidRequest) (string, error) {
			parsedURL, err := url.Parse(base)
			if err != nil {
				return "", fmt.Errorf("parse endpoint: %w", err)
			}

			query := parsedURL.Query()
			query.Set("account", a.Account)
			if req.Test == 1 {
				query.Set("testAdsEnabled", "true")
			}
			parsedURL.RawQuery = query.Encode()
			return parsedURL.String(), nil
		},
	})
}

// Builder constructs a bidder for Start.io based on processed configuration.
func Builder(cfg adapter.ProcessedConfigsMap, client *http.Client) (*adapters.Bidder, error) {
	startioCfg := cfg[adapter.StartIOKey]

	tagID, _ := startioCfg["tag_id"].(string)
	appID, _ := startioCfg["app_id"].(string)
	account, _ := startioCfg["account"].(string)

	adpt := &Adapter{
		TagID:   tagID,
		AppID:   appID,
		Account: account,
	}

	return &adapters.Bidder{
		Adapter: adpt,
		Client:  client,
	}, nil
}

func endpointByRegion(alpha3 string) string {
	switch geo.Region(alpha3) {
	case "asia":
		return "http://sin-trp-rtb.startappnetwork.com/1.3/2.5/getbid"
	case "eu":
		return "http://eu-trp-rtb.startappnetwork.com/1.3/2.5/getbid"
	default:
		return "http://trp-rtb.startappnetwork.com/1.3/2.5/getbid"
	}
}
