package taurusx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

type TaurusXAdapter struct {
	AppID string
	TagID string
}

var _ adapters.BidderInterface = (*TaurusXAdapter)(nil)

func (a *TaurusXAdapter) banner(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	imp, _ := adapters.BuildBannerImp(auctionRequest, adapters.BannerImpOptions{})
	return imp
}

func (a *TaurusXAdapter) interstitial(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	return adapters.BuildInterstitialImp(auctionRequest, adapters.InterstitialImpOptions{
		DisableOrientationSwap: true,
	})
}

func (a *TaurusXAdapter) rewarded(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	return adapters.BuildRewardedImp(auctionRequest, adapters.RewardedImpOptions{
		DisableOrientationSwap: true,
	})
}

func (a *TaurusXAdapter) BuildImpression(request openrtb.BidRequest, auctionRequest *schema.AuctionRequest) (*openrtb2.Imp, adapters.RTBRequestOptions, error) {
	if a.TagID == "" {
		return nil, adapters.RTBRequestOptions{}, errors.New("TagID is empty")
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

	opts := adapters.RTBRequestOptions{
		TagID: a.TagID,
	}
	if request.App != nil {
		opts.AppID = a.AppID
	}

	return imp, opts, nil
}

func (a *TaurusXAdapter) EnrichOpenRTBRequest(request *openrtb.BidRequest, auctionRequest *schema.AuctionRequest) error {
	reqExt := make(map[string]interface{})
	if request.Ext != nil {
		_ = json.Unmarshal(request.Ext, &reqExt)
	}

	if demandData, ok := auctionRequest.AdObject.Demands[adapter.TaurusXKey]; ok {
		if tokenData, ok := demandData["token"].(string); ok && tokenData != "" {
			placementToken, err := a.extractPlacementToken(tokenData, a.TagID)
			if err == nil && placementToken != "" {
				reqExt["token"] = placementToken
			}
		}
	}

	extBytes, err := json.Marshal(reqExt)
	if err != nil {
		return err
	}
	request.Ext = extBytes
	return nil
}

func (a *TaurusXAdapter) extractPlacementToken(tokenData, placementID string) (string, error) {
	if tokenData == "" || placementID == "" {
		return "", errors.New("empty token data or placement ID")
	}

	var tokenMap map[string]string
	err := json.Unmarshal([]byte(tokenData), &tokenMap)
	if err != nil {
		return "", fmt.Errorf("failed to parse token JSON: %v", err)
	}

	if placementToken, exists := tokenMap[placementID]; exists {
		return placementToken, nil
	}

	return "", fmt.Errorf("no token found for placement ID: %s", placementID)
}

func (a *TaurusXAdapter) ExecuteOptions(request openrtb.BidRequest) (adapters.ExecuteRTBOptions, error) {
	opts := adapters.ExecuteRTBOptions{
		TagID: a.TagID,
	}
	url := getEndpoint(adapters.CountryFromRequest(request))
	if url == "" {
		return opts, errors.New("taurusx endpoint is empty")
	}

	opts.URL = url
	opts.Headers = http.Header{"X-OpenRTB-Version": {"2.5"}}
	return opts, nil
}

// EnrichOpenRTBBid replaces Payload with BidResponse.ext.payload.
// TaurusX delivers the creative only there; bid.adm (already set by the
// shared parser) is deliberately ignored.
func (a *TaurusXAdapter) EnrichOpenRTBBid(
	dr *adapters.DemandResponse,
	bidResp *openrtb2.BidResponse,
	_ openrtb2.SeatBid,
	_ openrtb2.Bid,
) error {
	payload := ""
	if bidResp.Ext != nil {
		var extData map[string]interface{}
		if err := json.Unmarshal(bidResp.Ext, &extData); err != nil {
			return fmt.Errorf("failed to unmarshal bid response ext: %v", err)
		}
		if payloadValue, exists := extData["payload"]; exists {
			if payloadStr, ok := payloadValue.(string); ok {
				payload = payloadStr
			}
		}
	}
	dr.Bid.Payload = payload
	return nil
}

// Builder builds a new instance of the TaurusX adapter for the given bidder with the given config.
func Builder(cfg adapter.ProcessedConfigsMap, client *http.Client) (*adapters.Bidder, error) {
	tCfg := cfg[adapter.TaurusXKey]

	appID, ok := tCfg["app_id"].(string)
	if !ok || appID == "" {
		return nil, fmt.Errorf("missing app_id param for %s adapter", adapter.TaurusXKey)
	}
	tagID, ok := tCfg["tag_id"].(string)
	if !ok {
		tagID = ""
	}

	adpt := &TaurusXAdapter{
		AppID: appID,
		TagID: tagID,
	}

	bidder := adapters.Bidder{
		Adapter: adpt,
		Client:  client,
	}

	return &bidder, nil
}

// alpha3ToRegionMapping maps country codes to TaurusX regions
var alpha3ToRegionMapping = map[string]string{
	// US region countries (Americas)
	"ABW": "us", "AIA": "us", "ARG": "us", "ATG": "us", "BES": "us", "BHS": "us", "BLM": "us", "BLZ": "us",
	"BOL": "us", "BRA": "us", "BRB": "us", "CAN": "us", "CHL": "us", "COL": "us", "CRI": "us", "CUB": "us",
	"CUW": "us", "CYM": "us", "DMA": "us", "DOM": "us", "ECU": "us", "GLP": "us", "GRD": "us", "GRL": "us",
	"GTM": "us", "GUF": "us", "GUY": "us", "HND": "us", "HTI": "us", "JAM": "us", "KNA": "us", "LCA": "us",
	"MAF": "us", "MEX": "us", "MSR": "us", "MTQ": "us", "NIC": "us", "PAN": "us", "PER": "us", "PRI": "us",
	"PRY": "us", "SLV": "us", "SUR": "us", "SXM": "us", "TCA": "us", "TST": "us", "TTO": "us", "UMI": "us",
	"URY": "us", "USA": "us", "VCT": "us", "VEN": "us", "VGB": "us", "VIR": "us",

	// Asia region countries
	"AFG": "sg", "ARE": "sg", "ARM": "sg", "ASM": "sg", "ATA": "sg", "ATF": "sg", "AUS": "sg",
	"BGD": "sg", "BHR": "sg", "BRN": "sg", "BTN": "sg", "CCK": "sg", "CHN": "sg", "COK": "sg",
	"COM": "sg", "CXR": "sg", "FJI": "sg", "FSM": "sg", "GUM": "sg", "HKG": "sg", "HMD": "sg",
	"IDN": "sg", "IND": "sg", "IOT": "sg", "IRN": "sg", "IRQ": "sg", "ISR": "sg", "JPN": "sg",
	"KAZ": "sg", "KHM": "sg", "KIR": "sg", "KOR": "sg", "KWT": "sg", "LAO": "sg", "LBN": "sg",
	"LKA": "sg", "MAC": "sg", "MDV": "sg", "MHL": "sg", "MMR": "sg", "MNG": "sg", "MNP": "sg",
	"MYS": "sg", "MYT": "sg", "NCL": "sg", "NFK": "sg", "NIU": "sg", "NPL": "sg", "NRU": "sg",
	"NZL": "sg", "OMN": "sg", "PAK": "sg", "PCN": "sg", "PHL": "sg", "PLW": "sg", "PNG": "sg",
	"PRK": "sg", "PYF": "sg", "QAT": "sg", "SAU": "sg", "SGP": "sg", "SLB": "sg", "SSG": "sg",
	"SYC": "sg", "THA": "sg", "TJK": "sg", "TKL": "sg", "TKM": "sg", "TLS": "sg", "TON": "sg",
	"TUV": "sg", "TWN": "sg", "UZB": "sg", "VNM": "sg", "VUT": "sg", "WLF": "sg", "WSM": "sg",
	"YEM": "sg",

	// EU region countries (Europe, Africa, Middle East)
	"AGO": "eu", "ALA": "eu", "ALB": "eu", "AND": "eu", "AUT": "eu", "AZE": "eu", "BDI": "eu", "BEL": "eu",
	"BEN": "eu", "BFA": "eu", "BGR": "eu", "BIH": "eu", "BLR": "eu", "BMU": "eu", "BVT": "eu", "BWA": "eu",
	"CAF": "eu", "CHE": "eu", "CIV": "eu", "CMR": "eu", "COD": "eu", "COG": "eu", "CPV": "eu", "CYP": "eu",
	"CZE": "eu", "DEU": "eu", "DJI": "eu", "DNK": "eu", "DZA": "eu", "EGY": "eu", "ERI": "eu", "ESH": "eu",
	"ESP": "eu", "EST": "eu", "ETH": "eu", "FIN": "eu", "FRA": "eu", "FRO": "eu", "GAB": "eu", "GBR": "eu",
	"GEO": "eu", "GGY": "eu", "GHA": "eu", "GIB": "eu", "GIN": "eu", "GMB": "eu", "GNB": "eu", "GNQ": "eu",
	"GRC": "eu", "HRV": "eu", "HUN": "eu", "IMN": "eu", "IRL": "eu", "ISL": "eu", "ITA": "eu", "JEY": "eu",
	"JOR": "eu", "KEN": "eu", "KGZ": "eu", "LBR": "eu", "LBY": "eu", "LIE": "eu", "LSO": "eu", "LTU": "eu",
	"LUX": "eu", "LVA": "eu", "MAR": "eu", "MCO": "eu", "MDA": "eu", "MDG": "eu", "MKD": "eu", "MLI": "eu",
	"MLT": "eu", "MNE": "eu", "MOZ": "eu", "MRT": "eu", "MUS": "eu", "MWI": "eu", "NAM": "eu", "NER": "eu",
	"NGA": "eu", "NLD": "eu", "NOR": "eu", "POL": "eu", "PRT": "eu", "PSE": "eu", "REU": "eu", "ROU": "eu",
	"RUS": "eu", "RWA": "eu", "SDN": "eu", "SEN": "eu", "SGS": "eu", "SHN": "eu", "SJM": "eu", "SLE": "eu",
	"SMR": "eu", "SOM": "eu", "SRB": "eu", "SSD": "eu", "STP": "eu", "SVK": "eu", "SVN": "eu", "SWE": "eu",
	"SWZ": "eu", "SYR": "eu", "TCD": "eu", "TGO": "eu", "TUN": "eu", "TUR": "eu", "TZA": "eu", "UGA": "eu",
	"UKR": "eu", "VAT": "eu", "ZAF": "eu", "ZMB": "eu", "ZWE": "eu",
}

const defaultRegion = "us"

// getEndpoint returns the appropriate TaurusX endpoint based on country code
func getEndpoint(alpha3 string) string {
	// Determine region based on country code
	region := defaultRegion
	if mappedRegion, ok := alpha3ToRegionMapping[alpha3]; ok {
		region = mappedRegion
	}

	// Return the appropriate regional endpoint
	switch region {
	case "eu":
		return "https://sdkeu.ssp.taxssp.com/ssp/v1/bidding_ad/bidon"
	case "sg":
		return "https://sdksg.ssp.taxssp.com/ssp/v1/bidding_ad/bidon"
	case "us":
		fallthrough
	default:
		return "https://sdkus.ssp.taxssp.com/ssp/v1/bidding_ad/bidon"
	}
}
