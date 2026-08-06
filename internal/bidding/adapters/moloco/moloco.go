package moloco

import (
	"context"
	"errors"
	"net/http"

	"github.com/prebid/openrtb/v19/adcom1"
	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

// MolocoAdapter represents the Moloco bidding adapter
type MolocoAdapter struct { //nolint:revive
	TagID  string
	AppID  string
	APIKey string
}

var _ adapters.BidderInterface = (*MolocoAdapter)(nil)

// banner creates a banner impression for the bid request
func (a *MolocoAdapter) banner(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	imp, _ := adapters.BuildBannerImp(auctionRequest, adapters.BannerImpOptions{})
	return imp
}

// interstitial creates an interstitial impression for the bid request
func (a *MolocoAdapter) interstitial(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	return adapters.BuildInterstitialImp(auctionRequest, adapters.InterstitialImpOptions{
		IncludeVideo: true,
	})
}

// rewarded creates a rewarded video impression for the bid request
func (a *MolocoAdapter) rewarded(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	skip := int8(1)
	return adapters.BuildRewardedImp(auctionRequest, adapters.RewardedImpOptions{
		IncludeBanner: true,
		Rwdd:          1,
		Skip:          &skip,
		BannerBAttr:   []adcom1.CreativeAttribute{16},
		VideoBAttr:    []adcom1.CreativeAttribute{1, 2, 5, 8, 9, 14, 17},
	})
}

func (a *MolocoAdapter) BuildImpression(request openrtb.BidRequest, auctionRequest *schema.AuctionRequest) (*openrtb2.Imp, adapters.RTBRequestOptions, error) {
	if a.TagID == "" {
		return nil, adapters.RTBRequestOptions{}, errors.New("moloco AdUnitID is empty")
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
	// Preserve previous behavior: only set App.ID when App is already present.
	if a.AppID != "" && request.App != nil {
		opts.AppID = a.AppID
	}
	if token, ok := auctionRequest.AdObject.Demands[adapter.MolocoKey]["token"].(string); ok {
		opts.BuyerUID = token
	}

	return imp, opts, nil
}

// ExecuteRequest implements the BidderInterface.ExecuteRequest method
func (a *MolocoAdapter) ExecuteRequest(ctx context.Context, client *http.Client, request openrtb.BidRequest) *adapters.DemandResponse {
	url := getEndpoint(adapters.CountryFromRequest(request))
	if url == "" {
		return &adapters.DemandResponse{
			DemandID:  adapter.MolocoKey,
			RequestID: request.ID,
			TagID:     a.TagID,
			Error:     errors.New("moloco endpoint is empty"),
		}
	}
	if a.APIKey == "" {
		return &adapters.DemandResponse{
			DemandID:  adapter.MolocoKey,
			RequestID: request.ID,
			TagID:     a.TagID,
			Error:     errors.New("moloco API key is empty"),
		}
	}

	return adapters.ExecuteRTBRequest(ctx, client, request, adapters.ExecuteRTBOptions{
		DemandID: adapter.MolocoKey,
		URL:      url,
		TagID:    a.TagID,
		Headers:  http.Header{"Authorization": {a.APIKey}},
	})
}

// Builder builds a new instance of the Moloco adapter for the given bidder with the given config.
func Builder(cfg adapter.ProcessedConfigsMap, client *http.Client) (*adapters.Bidder, error) {
	molocoCfg := cfg[adapter.MolocoKey]

	tagID, ok := molocoCfg["tag_id"].(string)
	if !ok {
		tagID = ""
	}

	appID, ok := molocoCfg["app_id"].(string)
	if !ok {
		appID = ""
	}

	apiKey, ok := molocoCfg["api_key"].(string)
	if !ok {
		apiKey = ""
	}

	adpt := &MolocoAdapter{
		TagID:  tagID,
		AppID:  appID,
		APIKey: apiKey,
	}

	bidder := &adapters.Bidder{
		Adapter: adpt,
		Client:  client,
	}

	return bidder, nil
}

// alpha3ToRegionMapping maps country codes to Moloco regions based on country_mapping.json
var alpha3ToRegionMapping = map[string]string{
	// US region countries
	"ABW": "us", "AIA": "us", "ARG": "us", "ATG": "us", "BES": "us", "BHS": "us", "BLM": "us", "BLZ": "us",
	"BOL": "us", "BRA": "us", "BRB": "us", "CAN": "us", "CHL": "us", "COL": "us", "CRI": "us", "CUB": "us",
	"CUW": "us", "CYM": "us", "DMA": "us", "DOM": "us", "ECU": "us", "GLP": "us", "GRD": "us", "GRL": "us",
	"GTM": "us", "GUF": "us", "GUY": "us", "HND": "us", "HTI": "us", "JAM": "us", "KNA": "us", "LCA": "us",
	"MAF": "us", "MEX": "us", "MSR": "us", "MTQ": "us", "NIC": "us", "PAN": "us", "PER": "us", "PRI": "us",
	"PRY": "us", "SLV": "us", "SUR": "us", "SXM": "us", "TCA": "us", "TST": "us", "TTO": "us", "UMI": "us",
	"URY": "us", "USA": "us", "VCT": "us", "VEN": "us", "VGB": "us", "VIR": "us",

	// Asia region countries
	"AFG": "asia", "ARE": "asia", "ARM": "asia", "ASM": "asia", "ATA": "asia", "ATF": "asia", "AUS": "asia",
	"BGD": "asia", "BHR": "asia", "BRN": "asia", "BTN": "asia", "CCK": "asia", "CHN": "asia", "COK": "asia",
	"COM": "asia", "CXR": "asia", "FJI": "asia", "FSM": "asia", "GUM": "asia", "HKG": "asia", "HMD": "asia",
	"IDN": "asia", "IND": "asia", "IOT": "asia", "IRN": "asia", "IRQ": "asia", "ISR": "asia", "JPN": "asia",
	"KAZ": "asia", "KHM": "asia", "KIR": "asia", "KOR": "asia", "KWT": "asia", "LAO": "asia", "LBN": "asia",
	"LKA": "asia", "MAC": "asia", "MDV": "asia", "MHL": "asia", "MMR": "asia", "MNG": "asia", "MNP": "asia",
	"MYS": "asia", "MYT": "asia", "NCL": "asia", "NFK": "asia", "NIU": "asia", "NPL": "asia", "NRU": "asia",
	"NZL": "asia", "OMN": "asia", "PAK": "asia", "PCN": "asia", "PHL": "asia", "PLW": "asia", "PNG": "asia",
	"PRK": "asia", "PYF": "asia", "QAT": "asia", "SAU": "asia", "SGP": "asia", "SLB": "asia", "SSG": "asia",
	"SYC": "asia", "THA": "asia", "TJK": "asia", "TKL": "asia", "TKM": "asia", "TLS": "asia", "TON": "asia",
	"TUV": "asia", "TWN": "asia", "UZB": "asia", "VNM": "asia", "VUT": "asia", "WLF": "asia", "WSM": "asia",
	"YEM": "asia",

	// EU region countries
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

// getEndpoint returns the appropriate Moloco endpoint based on country code and fallback endpoint
func getEndpoint(alpha3 string) string {
	// Determine region based on country code
	region := defaultRegion
	if mappedRegion, ok := alpha3ToRegionMapping[alpha3]; ok {
		region = mappedRegion
	}

	return "https://sdkfnt-" + region + ".dsp-api.moloco.com/mediations/inhouse/v1"
}
