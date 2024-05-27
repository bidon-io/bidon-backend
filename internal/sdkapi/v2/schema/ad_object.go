package schema

import (
	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
)

type AdObject struct {
	AuctionID               string                         `json:"auction_id" validate:"required,uuid4"`
	AuctionConfigurationID  int64                          `json:"auction_configuration_id"`
	AuctionConfigurationUID string                         `json:"auction_configuration_uid"`
	PriceFloor              float64                        `json:"pricefloor" validate:"required,gte=0"`
	Orientation             string                         `json:"orientation" validate:"oneof=PORTRAIT LANDSCAPE"`
	Demands                 map[adapter.Key]map[string]any `json:"demands"`
	Banner                  *BannerAdObject                `json:"banner"`
	Interstitial            *InterstitialAdObject          `json:"interstitial"`
	Rewarded                *RewardedAdObject              `json:"rewarded"`
}

func (o *AdObject) ToImp(roundID string) Imp {
	return Imp{
		AuctionID:               o.AuctionID,
		AuctionConfigurationID:  o.AuctionConfigurationID,
		AuctionConfigurationUID: o.AuctionConfigurationUID,
		RoundID:                 roundID,
		BidFloor:                &o.PriceFloor,
		Orientation:             o.Orientation,
		Demands:                 o.Demands,
		Banner:                  o.Banner,
		Interstitial:            o.Interstitial,
		Rewarded:                o.Rewarded,
	}
}

func (o *AdObject) Format() ad.Format {
	if o.Banner != nil {
		return o.Banner.Format
	}

	return ad.EmptyFormat
}

type BannerAdObject struct {
	Format ad.Format `json:"format" validate:"oneof=BANNER LEADERBOARD MREC ADAPTIVE"`
}

func (o BannerAdObject) Map() map[string]any {
	return map[string]any{
		"format": o.Format,
	}
}

type InterstitialAdObject struct{}

func (o InterstitialAdObject) Map() map[string]any {
	return map[string]any{}
}

type RewardedAdObject struct{}

func (o RewardedAdObject) Map() map[string]any {
	return map[string]any{}
}
