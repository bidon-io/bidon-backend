//go:build e2e

package e2e

import (
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/require"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/device"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

// Response shapes are re-declared locally rather than imported from
// apihandlers/auction, because ConfigResponseInit.Adapters is keyed by a
// per-adapter interface with no custom unmarshaler — decoding it generically
// is simpler and is all these tests need.

type configResponse struct {
	Init struct {
		Adapters map[string]any `json:"adapters"`
	} `json:"init"`
}

type auctionResponse struct {
	ConfigID                 int64    `json:"auction_configuration_id"`
	ConfigUID                string   `json:"auction_configuration_uid"`
	ExternalWinNotifications bool     `json:"external_win_notifications"`
	AdUnits                  []adUnit `json:"ad_units"`
	NoBids                   []adUnit `json:"no_bids"`
	AuctionID                string   `json:"auction_id"`
}

type adUnit struct {
	DemandID   string         `json:"demand_id"`
	UID        string         `json:"uid"`
	Label      string         `json:"label"`
	PriceFloor *float64       `json:"pricefloor"`
	BidType    string         `json:"bid_type"`
	Ext        map[string]any `json:"ext"`
}

func newDevice() schema.Device {
	js := 1
	return schema.Device{
		Manufacturer:    "Google",
		Model:           "Pixel 3",
		OS:              "android",
		OSVersion:       "12",
		HardwareVersion: "blueline",
		Height:          2028,
		Width:           1080,
		PPI:             440,
		PXRatio:         3.0,
		JS:              &js,
		Language:        "en_US",
		ConnectionType:  "WIFI",
		Type:            device.PhoneType,
		UserAgent:       "Mozilla/5.0 (Linux; Android 12; Pixel 3) bidon-e2e",
	}
}

func newSession(t *testing.T) schema.Session {
	t.Helper()

	id, err := uuid.NewV4()
	require.NoError(t, err)

	cpuUsage := 0.5
	return schema.Session{
		ID:                        id.String(),
		LaunchTS:                  1700000000000,
		LaunchMonotonicTS:         1000,
		StartTS:                   1700000000000,
		StartMonotonicTS:          1000,
		TS:                        1700000001000,
		MonotonicTS:               2000,
		MemoryWarningsTS:          []int{},
		MemoryWarningsMonotonicTS: []int{},
		RAMUsed:                   100000000,
		RAMSize:                   2000000000,
		Battery:                   1.0,
		CPUUsage:                  &cpuUsage,
	}
}

func newUser(t *testing.T) schema.User {
	t.Helper()

	idfa, err := uuid.NewV4()
	require.NoError(t, err)
	idg, err := uuid.NewV4()
	require.NoError(t, err)

	return schema.User{
		IDFA:                        idfa.String(),
		IDG:                         idg.String(),
		TrackingAuthorizationStatus: "AUTHORIZED",
	}
}

func newSchemaApp(f fixture) schema.App {
	return schema.App{
		Bundle:    f.app.PackageName.String,
		Key:       f.app.AppKey.String,
		Framework: "native",
		Version:   "1.0",
	}
}

func newBaseRequest(t *testing.T, f fixture) schema.BaseRequest {
	t.Helper()

	return schema.BaseRequest{
		Device:      newDevice(),
		Session:     newSession(t),
		App:         newSchemaApp(f),
		User:        newUser(t),
		Regulations: &schema.Regulations{},
	}
}

func newAdikteevAdapters() schema.Adapters {
	return schema.Adapters{
		adapter.AdikteevKey: schema.Adapter{Version: "1", SDKVersion: "11.8.2"},
	}
}

func newConfigRequest(t *testing.T, f fixture) schema.ConfigRequest {
	t.Helper()

	return schema.ConfigRequest{
		BaseRequest: newBaseRequest(t, f),
		Adapters:    newAdikteevAdapters(),
	}
}

// newAuctionRequest builds a rewarded auction request bidding for floor, with
// a non-empty Adikteev token so the bidding builder actually calls out to
// dspsim (internal/bidding/builder.go's 4-way adapter intersection).
func newAuctionRequest(t *testing.T, f fixture, floor float64) schema.AuctionRequest {
	t.Helper()

	auctionID, err := uuid.NewV4()
	require.NoError(t, err)

	return schema.AuctionRequest{
		BaseRequest: newBaseRequest(t, f),
		Adapters:    newAdikteevAdapters(),
		AdObject: schema.AdObject{
			AuctionID:   auctionID.String(),
			PriceFloor:  floor,
			Orientation: "PORTRAIT",
			Demands: map[adapter.Key]map[string]any{
				adapter.AdikteevKey: {"token": "sdk-instance-id"},
			},
			Rewarded: &schema.RewardedAdObject{},
		},
		TMax: 2000,
	}
}

// newBid builds the schema.Bid the SDK would echo back from an ad_units[]
// entry of an auction response: same demand, same price, same auction
// configuration uid.
func newBid(auctionConfigUID, auctionID string, au adUnit) *schema.Bid {
	price := au.PriceFloor
	var p float64
	if price != nil {
		p = *price
	}
	return &schema.Bid{
		AuctionID:               auctionID,
		AuctionConfigurationUID: auctionConfigUID,
		DemandID:                au.DemandID,
		AdUnitUID:               au.UID,
		Price:                   p,
		BidType:                 schema.RTBBidType,
		Rewarded:                &schema.RewardedAdObject{},
	}
}

func newWinRequest(t *testing.T, f fixture, bid *schema.Bid) schema.WinRequest {
	t.Helper()

	return schema.WinRequest{
		ShowRequest: schema.ShowRequest{
			BaseRequest: newBaseRequest(t, f),
			Bid:         bid,
		},
	}
}

func newShowRequest(t *testing.T, f fixture, bid *schema.Bid) schema.ShowRequest {
	t.Helper()

	return schema.ShowRequest{
		BaseRequest: newBaseRequest(t, f),
		Bid:         bid,
	}
}

func newLossRequest(t *testing.T, f fixture, bid *schema.Bid, externalWinnerPrice float64) schema.LossRequest {
	t.Helper()

	price := externalWinnerPrice
	return schema.LossRequest{
		ShowRequest: schema.ShowRequest{
			BaseRequest: newBaseRequest(t, f),
			Bid:         bid,
		},
		ExternalWinner: schema.ExternalWinner{
			DemandID: "external_network",
			Price:    &price,
		},
	}
}
