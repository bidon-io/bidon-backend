package auction

import (
	"testing"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/bidding/rendering"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

func TestConvertBidToAdUnit_PropagatesRendering(t *testing.T) {
	storeAdUnit := AdUnit{
		DemandID: string(adapter.InmobiKey),
		UID:      "uid-1",
		Label:    "label-1",
		BidType:  schema.RTBBidType,
		Timeout:  30,
	}
	adUnitsMap := buildAdUnitsMap(&[]AdUnit{storeAdUnit})

	renderingCfg := &rendering.Config{
		Creative: &rendering.CreativeConfig{Type: rendering.CreativeTypeVAST},
	}
	demandResponse := adapters.DemandResponse{
		DemandID: adapter.InmobiKey,
		Bid: &adapters.NormalizedBid{
			DemandID:  adapter.InmobiKey,
			Price:     1.5,
			Rendering: renderingCfg,
		},
	}

	got := convertBidToAdUnit(&schema.AuctionRequest{}, demandResponse, adUnitsMap)

	if got == nil {
		t.Fatal("convertBidToAdUnit returned nil")
	}
	if got.Rendering != renderingCfg {
		t.Fatalf("Rendering = %+v, want the same pointer as the demand response bid's Rendering (%+v)", got.Rendering, renderingCfg)
	}
}

func TestConvertBidToAdUnit_NoBidLeavesRenderingNil(t *testing.T) {
	storeAdUnit := AdUnit{
		DemandID: string(adapter.InmobiKey),
		UID:      "uid-1",
		Label:    "label-1",
		BidType:  schema.RTBBidType,
		Timeout:  30,
	}
	adUnitsMap := buildAdUnitsMap(&[]AdUnit{storeAdUnit})

	demandResponse := adapters.DemandResponse{
		DemandID: adapter.InmobiKey,
	}

	got := convertBidToAdUnit(&schema.AuctionRequest{}, demandResponse, adUnitsMap)

	if got == nil {
		t.Fatal("convertBidToAdUnit returned nil")
	}
	if got.Rendering != nil {
		t.Fatalf("Rendering = %+v, want nil for a no-bid demand response", got.Rendering)
	}
}
