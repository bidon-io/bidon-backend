package mobilefuse

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"

	"github.com/prebid/openrtb/v19/adcom1"
	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

var (
	// ErrUnsupportedRegion indicates that the bid request is from a geographic region
	// not supported by MobileFuse. According to OpenRTB specification, this corresponds
	// to NoBidReason.UNSUPPORTED_DEVICE (code 6) for geographic restrictions.
	ErrUnsupportedRegion = errors.New("unsupported device")
)

// supportedCountries defines the ISO-3166-1-alpha-3 country codes supported by MobileFuse
var supportedCountries = []string{"USA", "CAN"}

// newUnsupportedRegionError creates an enhanced error message with contextual information
func newUnsupportedRegionError(received string) error {
	if received == "" {
		received = "none"
	}
	return fmt.Errorf("%w: received '%s', expected 'USA' or 'CAN'", ErrUnsupportedRegion, received)
}

type MobileFuseAdapter struct {
	TagID string
}

var _ adapters.BidderInterface = (*MobileFuseAdapter)(nil)

func (a *MobileFuseAdapter) banner(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	imp, _ := adapters.BuildBannerImp(auctionRequest, adapters.BannerImpOptions{})
	return imp
}

// interstitial stays custom: Pos-only fullscreen Banner without W/H.
func (a *MobileFuseAdapter) interstitial() *openrtb2.Imp {
	return &openrtb2.Imp{
		Instl: 1,
		Banner: &openrtb2.Banner{
			Pos: adcom1.PositionFullScreen.Ptr(),
		},
	}
}

// rewarded stays custom: minimal mp4 Video without sized fullscreen defaults.
func (a *MobileFuseAdapter) rewarded() *openrtb2.Imp {
	return &openrtb2.Imp{
		Instl: 0,
		Video: &openrtb2.Video{
			MIMEs: []string{"video/mp4"},
		},
	}
}

func (a *MobileFuseAdapter) BuildImpression(request openrtb.BidRequest, auctionRequest *schema.AuctionRequest) (*openrtb2.Imp, adapters.RTBRequestOptions, error) {
	if a.TagID == "" {
		return nil, adapters.RTBRequestOptions{}, errors.New("TagID is empty")
	}

	country := ""
	if request.Device != nil && request.Device.Geo != nil {
		country = request.Device.Geo.Country
	}

	if !slices.Contains(supportedCountries, country) {
		return nil, adapters.RTBRequestOptions{}, newUnsupportedRegionError(country)
	}

	var imp *openrtb2.Imp
	switch auctionRequest.AdObject.Type() {
	case ad.BannerType:
		imp = a.banner(auctionRequest)
	case ad.InterstitialType:
		imp = a.interstitial()
	case ad.RewardedType:
		imp = a.rewarded()
	default:
		return nil, adapters.RTBRequestOptions{}, errors.New("unknown impression type")
	}

	return imp, adapters.RTBRequestOptions{
		TagID:           a.TagID,
		OmitBidFloorCur: true,
	}, nil
}

func (a *MobileFuseAdapter) EnrichOpenRTBRequest(request *openrtb.BidRequest, auctionRequest *schema.AuctionRequest) error {
	token, _ := auctionRequest.AdObject.Demands[adapter.MobileFuseKey]["token"].(string)
	request.User = &openrtb.User{
		Data: []openrtb.Data{
			{
				Segment: []openrtb.Segment{
					{
						Signal: token,
					},
				},
			},
		},
	}
	return nil
}

func (a *MobileFuseAdapter) ExecuteRequest(ctx context.Context, client *http.Client, request openrtb.BidRequest) *adapters.DemandResponse {
	dr := &adapters.DemandResponse{
		DemandID:  adapter.MobileFuseKey,
		RequestID: request.ID,
		TagID:     a.TagID,
	}
	requestBody, err := json.Marshal(request)
	if err != nil {
		dr.Error = err
		return dr
	}
	dr.RawRequest = string(requestBody)

	url := "https://mfx.mobilefuse.com/openrtb?ssp=bidon"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(requestBody))
	if err != nil {
		dr.Error = err
		return dr
	}
	httpReq.Header.Add("Content-Type", "application/json")

	httpResp, err := client.Do(httpReq)
	if err != nil {
		dr.Error = err
		return dr
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		dr.Error = err
		return dr
	}

	dr.RawResponse = string(respBody)
	dr.Status = httpResp.StatusCode

	return dr
}

// Builder builds a new instance of the MobileFuse adapter for the given bidder with the given config.
func Builder(cfg adapter.ProcessedConfigsMap, client *http.Client) (*adapters.Bidder, error) {
	mobileFuseCfg := cfg[adapter.MobileFuseKey]

	tagID, ok := mobileFuseCfg["tag_id"].(string)
	if !ok {
		tagID = ""
	}

	adpt := &MobileFuseAdapter{
		TagID: tagID,
	}

	bidder := &adapters.Bidder{
		Adapter: adpt,
		Client:  client,
	}

	return bidder, nil
}
