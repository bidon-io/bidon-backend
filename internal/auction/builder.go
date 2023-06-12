package auction

import (
	"context"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/device"
	"golang.org/x/exp/slices"
)

type Builder struct {
	ConfigMatcher   ConfigMatcher
	LineItemMatcher LineItemsMatcher
}

type ConfigMatcher interface {
	Match(ctx context.Context, appID int64, adType ad.Type) (*Config, error)
}

type LineItemsMatcher interface {
	Match(ctx context.Context, appID int64, adType ad.Type, adFormats []ad.Format, adapters []string) ([]LineItem, error)
}

func (b *Builder) Build(ctx context.Context, params *BuildParams) (*Auction, error) {
	config, err := b.ConfigMatcher.Match(ctx, params.AppID, params.AdType)
	if err != nil {
		return nil, err
	}

	adFormats := resolveAdFormats(params.AdType, params.AdFormat, params.DeviceType)
	lineItems, err := b.LineItemMatcher.Match(ctx, params.AppID, params.AdType, adFormats, params.Adapters)
	if err != nil {
		return nil, err
	}

	auction := Auction{
		Rounds:     filterRounds(config.Rounds, params.Adapters),
		LineItems:  lineItems,
		Token:      "{}",
		PriceFloor: params.PriceFloor,
		ConfigID:   config.ID,
	}

	return &auction, nil
}

func filterRounds(rounds []RoundConfig, adapters []string) []RoundConfig {
	filteredRounds := []RoundConfig{}

	for _, round := range rounds {
		filteredDemands := []string{}
		for _, demand := range round.Demands {
			if slices.Contains(adapters, demand) {
				filteredDemands = append(filteredDemands, demand)
			}
		}
		if len(filteredDemands) == 0 {
			continue
		}

		round.Demands = filteredDemands
		filteredRounds = append(filteredRounds, round)
	}

	return filteredRounds
}

func resolveAdFormats(adType ad.Type, adFormat ad.Format, deviceType device.Type) (adFormats []ad.Format) {
	if adType != ad.BannerType {
		return
	}
	if !slices.Contains(ad.BannerFormats, adFormat) {
		return
	}

	adFormats = append(adFormats, adFormat)

	if adFormat != ad.AdaptiveFormat {
		return
	}

	if deviceType == device.TabletType {
		adFormats = append(adFormats, ad.LeaderboardFormat)
	} else {
		adFormats = append(adFormats, ad.BannerFormat)
	}

	return
}
