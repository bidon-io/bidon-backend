package bidmachine

import (
	"encoding/json"
	"errors"
	"fmt"
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

type BidmachineAdapter struct {
	SellerID string
	Endpoint string
}

var _ adapters.BidderInterface = (*BidmachineAdapter)(nil)

func (a *BidmachineAdapter) banner(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	imp, _ := adapters.BuildBannerImp(auctionRequest, adapters.BannerImpOptions{})
	return imp
}

func (a *BidmachineAdapter) interstitial(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	return adapters.BuildInterstitialImp(auctionRequest, adapters.InterstitialImpOptions{
		IncludeVideo: true,
	})
}

func (a *BidmachineAdapter) rewarded(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	return adapters.BuildRewardedImp(auctionRequest, adapters.RewardedImpOptions{
		IncludeBanner: true,
		BannerBAttr:   []adcom1.CreativeAttribute{16},
		VideoBAttr:    []adcom1.CreativeAttribute{16},
		ImpExt:        json.RawMessage(`{"rewarded": 1}`),
	})
}

func (a *BidmachineAdapter) BuildImpression(_ openrtb.BidRequest, auctionRequest *schema.AuctionRequest) (*openrtb2.Imp, adapters.RTBRequestOptions, error) {
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

	extStructure := &map[string]interface{}{}
	_ = json.Unmarshal(imp.Ext, extStructure)
	(*extStructure)["bid_token"] = auctionRequest.AdObject.Demands[adapter.BidmachineKey]["token"]
	raw, _ := json.Marshal(extStructure)
	imp.Ext = raw

	return imp, adapters.RTBRequestOptions{
		PublisherID:     a.SellerID,
		OmitBidFloorCur: true,
	}, nil
}

func (a *BidmachineAdapter) EnrichOpenRTBRequest(request *openrtb.BidRequest, auctionRequest *schema.AuctionRequest) error {
	ext, err := json.Marshal(ExtraParams(auctionRequest))
	if err != nil {
		return err
	}
	request.Ext = ext
	return nil
}

func (a *BidmachineAdapter) ExecuteOptions(request openrtb.BidRequest) (adapters.ExecuteRTBOptions, error) {
	return adapters.ExecuteRTBOptions{
		URL: getEndpoint(adapters.CountryFromRequest(request)),
	}, nil
}

// Builder builds a new instance of the Bidmachine adapter for the given bidder with the given config.
func Builder(cfg adapter.ProcessedConfigsMap, client *http.Client) (*adapters.Bidder, error) {
	bmCfg := cfg[adapter.BidmachineKey]
	endpoint, ok := bmCfg["endpoint"].(string)
	if !ok || endpoint == "" {
		return nil, fmt.Errorf("missing endpoint param for BM adapter")
	}
	sellerID, ok := bmCfg["seller_id"].(string)
	if !ok || sellerID == "" {
		return nil, fmt.Errorf("missing seller_id param for %s adapter", adapter.BidmachineKey)
	}

	adpt := &BidmachineAdapter{
		Endpoint: endpoint,
		SellerID: sellerID,
	}

	bidder := &adapters.Bidder{
		Adapter: adpt,
		Client:  client,
	}

	return bidder, nil
}

func ExtraParams(req *schema.AuctionRequest) map[string]any {
	customParameters := map[string]any{
		"bidon_sdk_version": req.App.SDKVersion,
	}
	if req.GetMediator() != "" {
		customParameters["mediator"] = req.GetMediator()
	}
	if existingExtra, ok := req.GetNestedExtData()["bidmachine"].(map[string]any); ok {
		for key, value := range existingExtra {
			customParameters[key] = value
		}
	}
	if _, ok := customParameters["mediation_mode"]; !ok {
		if slices.Contains(adapter.CustomAdapters[:], req.GetMediator()) {
			customParameters["mediation_mode"] = "bidon_ca"
		} else {
			customParameters["mediation_mode"] = "bidon"
		}
	}

	return customParameters
}

var alpha3ToDcMapping = map[string]string{
	"AFG": "apac",
	"AUS": "apac",
	"BHR": "apac",
	"BGD": "apac",
	"BTN": "apac",
	"BRN": "apac",
	"KHM": "apac",
	"CHN": "apac",
	"TLS": "apac",
	"FJI": "apac",
	"HKG": "apac",
	"IND": "apac",
	"IDN": "apac",
	"IRN": "apac",
	"IRQ": "apac",
	"ISR": "apac",
	"JPN": "apac",
	"JOR": "apac",
	"KAZ": "apac",
	"KGZ": "apac",
	"KWT": "apac",
	"LAO": "apac",
	"LBN": "apac",
	"MYS": "apac",
	"MDV": "apac",
	"MNG": "apac",
	"MMR": "apac",
	"NPL": "apac",
	"NZL": "apac",
	"PRK": "apac",
	"OMN": "apac",
	"PAK": "apac",
	"PNG": "apac",
	"PHL": "apac",
	"QAT": "apac",
	"SAU": "apac",
	"SGP": "apac",
	"KOR": "apac",
	"LKA": "apac",
	"SYR": "apac",
	"TWN": "apac",
	"TJK": "apac",
	"THA": "apac",
	"TKM": "apac",
	"ARE": "apac",
	"UZB": "apac",
	"VNM": "apac",
	"YEM": "apac",
	"AGL": "us",
	"ARG": "us",
	"BHS": "us",
	"BRB": "us",
	"BLZ": "us",
	"BMU": "us",
	"BOL": "us",
	"BRA": "us",
	"CAN": "us",
	"CYM": "us",
	"CHL": "us",
	"COL": "us",
	"CRI": "us",
	"CUB": "us",
	"DMA": "us",
	"DOM": "us",
	"ECU": "us",
	"SLV": "us",
	"GRD": "us",
	"GTM": "us",
	"GUY": "us",
	"HTI": "us",
	"HND": "us",
	"JAM": "us",
	"MEX": "us",
	"NIC": "us",
	"PAN": "us",
	"PRY": "us",
	"PER": "us",
	"PRI": "us",
	"KNA": "us",
	"LCA": "us",
	"VCT": "us",
	"SUR": "us",
	"TTO": "us",
	"URY": "us",
	"USA": "us",
	"VEN": "us",
}

const defaultDc = "eu"

func getEndpoint(alpha3 string) string {
	dc := defaultDc
	if rewrittenDc, ok := alpha3ToDcMapping[alpha3]; ok {
		dc = rewrittenDc
	}

	return "https://api-" + dc + ".bidmachine.io/auction/prebid/bidon"
}
