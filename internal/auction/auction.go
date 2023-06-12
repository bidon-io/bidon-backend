package auction

import (
	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/device"
)

type Config struct {
	ID     int64
	Rounds []RoundConfig
}

type BuildParams struct {
	AppID      int64
	AdType     ad.Type
	AdFormat   ad.Format
	DeviceType device.Type
	PriceFloor float64
	Adapters   []string
}

type Auction struct {
	ConfigID   int64         `json:"config_id"`
	Rounds     []RoundConfig `json:"rounds"`
	LineItems  []LineItem    `json:"line_items"`
	Token      string        `json:"token"`
	PriceFloor float64       `json:"pricefloor"`
}

type RoundConfig struct {
	Demands []string
}

type LineItem struct {
	ID         string  `json:"id"`
	PriceFloor float64 `json:"pricefloor"`
	AdUnitID   string  `json:"ad_unit_id"`
}
