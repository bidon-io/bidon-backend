package auction

import (
	"encoding/json"
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
		Bid: &adapters.DemandBid{
			DemandID:  adapter.InmobiKey,
			Price:     1.5,
			Rendering: renderingCfg,
		},
	}

	got := convertBidToAdUnit(&schema.AuctionRequest{}, demandResponse, adUnitsMap)

	if got == nil {
		t.Fatal("convertBidToAdUnit returned nil")
	}
	if got.Extra["rendering"] != renderingCfg {
		t.Fatalf("ext.rendering = %+v, want the same pointer as the demand response bid's Rendering", got.Extra["rendering"])
	}
}

func TestConvertBidToAdUnit_NestsRenderingInsideExtJSON(t *testing.T) {
	storeAdUnit := AdUnit{
		DemandID: string(adapter.InmobiKey),
		UID:      "uid-1",
		Label:    "label-1",
		BidType:  schema.RTBBidType,
		Timeout:  30,
		Extra:    map[string]any{"payload": "<ad>"},
	}
	adUnitsMap := buildAdUnitsMap(&[]AdUnit{storeAdUnit})

	demandResponse := adapters.DemandResponse{
		DemandID: adapter.InmobiKey,
		Bid: &adapters.DemandBid{
			DemandID: adapter.InmobiKey,
			Price:    1.5,
			Payload:  "<ad>",
			Rendering: &rendering.Config{
				Creative: &rendering.CreativeConfig{Type: rendering.CreativeTypeVAST},
			},
		},
	}

	got := convertBidToAdUnit(&schema.AuctionRequest{}, demandResponse, adUnitsMap)
	if got == nil {
		t.Fatal("convertBidToAdUnit returned nil")
	}

	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal ad unit: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal ad unit: %v", err)
	}
	if _, ok := payload["rendering"]; ok {
		t.Fatalf("rendering must sit inside ext, not as a sibling; got %s", raw)
	}
	ext, ok := payload["ext"].(map[string]any)
	if !ok {
		t.Fatalf("ext missing or not an object: %s", raw)
	}
	renderingObj, ok := ext["rendering"].(map[string]any)
	if !ok {
		t.Fatalf("ext.rendering missing or not an object: %s", raw)
	}
	creative, ok := renderingObj["creative"].(map[string]any)
	if !ok {
		t.Fatalf("ext.rendering.creative missing: %s", raw)
	}
	if creative["type"] != "vast" {
		t.Fatalf("ext.rendering.creative.type = %v, want vast", creative["type"])
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
	if _, ok := got.Extra["rendering"]; ok {
		t.Fatalf("ext.rendering = %+v, want absent for a no-bid demand response", got.Extra["rendering"])
	}
}
