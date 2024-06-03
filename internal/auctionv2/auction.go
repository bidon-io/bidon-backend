package auctionv2

import (
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/auction"
)

type Config struct {
	ID                       int64
	UID                      string
	ExternalWinNotifications bool
	Demands                  []adapter.Key
	Bidding                  []adapter.Key
	AdUnits                  []auction.AdUnit
	PriceFloor               float64
	Timeout                  int
}
