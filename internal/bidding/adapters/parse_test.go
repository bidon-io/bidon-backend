package adapters_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

type stubAdapter struct{}

func (stubAdapter) CreateRequest(openrtb.BidRequest, *schema.AuctionRequest) (openrtb.BidRequest, error) {
	return openrtb.BidRequest{}, nil
}

func (stubAdapter) ExecuteRequest(context.Context, *http.Client, openrtb.BidRequest) *adapters.DemandResponse {
	return nil
}

type enricherAdapter struct {
	stubAdapter
	called bool
}

func (a *enricherAdapter) EnrichOpenRTBBid(
	dr *adapters.DemandResponse,
	_ *openrtb2.BidResponse,
	_ openrtb2.SeatBid,
	_ openrtb2.Bid,
) error {
	a.called = true
	dr.Bid.Signaldata = "enriched"
	return nil
}

type customParserAdapter struct {
	stubAdapter
}

func (customParserAdapter) ParseBids(dr *adapters.DemandResponse) (*adapters.DemandResponse, error) {
	dr.Bid = &adapters.BidDemandResponse{
		DemandID: adapter.ZmaticooKey,
		Payload:  "custom",
		Price:    1.23,
	}
	return dr, nil
}

func TestParseDemandResponse_mapsOpenRTBBid(t *testing.T) {
	raw, _ := json.Marshal(openrtb2.BidResponse{
		ID: "resp-1",
		SeatBid: []openrtb2.SeatBid{{
			Seat: "seat-1",
			Bid: []openrtb2.Bid{{
				ID:    "bid-1",
				ImpID: "imp-1",
				Price: 2.5,
				AdM:   "<ad>",
				AdID:  "ad-1",
				LURL:  "https://l",
				NURL:  "https://n",
				BURL:  "https://b",
			}},
		}},
	})

	dr := &adapters.DemandResponse{
		DemandID:    adapter.VungleKey,
		Status:      http.StatusOK,
		RawResponse: string(raw),
	}

	got, err := adapters.ParseDemandResponse(stubAdapter{}, dr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Bid == nil {
		t.Fatal("expected bid")
	}
	bid := got.Bid
	if bid.ID != "bid-1" || bid.ImpID != "imp-1" || bid.Price != 2.5 || bid.Payload != "<ad>" {
		t.Fatalf("unexpected bid fields: %+v", bid)
	}
	if bid.DemandID != adapter.VungleKey || bid.AdID != "ad-1" || bid.SeatID != "seat-1" {
		t.Fatalf("unexpected ids: %+v", bid)
	}
	if bid.LURL != "https://l" || bid.NURL != "https://n" || bid.BURL != "https://b" {
		t.Fatalf("unexpected urls: %+v", bid)
	}
}

func TestParseDemandResponse_emptySeatBidIsNoBid(t *testing.T) {
	raw, _ := json.Marshal(openrtb2.BidResponse{ID: "resp-1", SeatBid: []openrtb2.SeatBid{}})
	dr := &adapters.DemandResponse{
		DemandID:    adapter.MolocoKey,
		Status:      http.StatusOK,
		RawResponse: string(raw),
	}

	got, err := adapters.ParseDemandResponse(stubAdapter{}, dr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Bid != nil {
		t.Fatalf("expected no bid, got %+v", got.Bid)
	}
}

func TestParseDemandResponse_noContent(t *testing.T) {
	dr := &adapters.DemandResponse{Status: http.StatusNoContent}
	got, err := adapters.ParseDemandResponse(stubAdapter{}, dr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Bid != nil {
		t.Fatalf("expected no bid")
	}
}

func TestParseDemandResponse_unauthorizedStatuses(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusServiceUnavailable} {
		_, err := adapters.ParseDemandResponse(stubAdapter{}, &adapters.DemandResponse{Status: status})
		if err == nil {
			t.Fatalf("expected error for status %d", status)
		}
	}
}

func TestParseDemandResponse_invalidJSON(t *testing.T) {
	_, err := adapters.ParseDemandResponse(stubAdapter{}, &adapters.DemandResponse{
		Status:      http.StatusOK,
		RawResponse: "{not-json",
	})
	if err == nil {
		t.Fatal("expected JSON error")
	}
}

func TestParseDemandResponse_callsEnricher(t *testing.T) {
	raw, _ := json.Marshal(openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{{
			Bid: []openrtb2.Bid{{ID: "bid-1", ImpID: "imp-1", Price: 1}},
		}},
	})
	enricher := &enricherAdapter{}
	dr := &adapters.DemandResponse{
		DemandID:    adapter.YandexKey,
		Status:      http.StatusOK,
		RawResponse: string(raw),
	}

	got, err := adapters.ParseDemandResponse(enricher, dr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !enricher.called {
		t.Fatal("expected enricher to be called")
	}
	if got.Bid.Signaldata != "enriched" {
		t.Fatalf("expected enriched signaldata, got %q", got.Bid.Signaldata)
	}
}

func TestParseDemandResponse_customParser(t *testing.T) {
	dr := &adapters.DemandResponse{Status: http.StatusOK, RawResponse: "ignored"}
	got, err := adapters.ParseDemandResponse(customParserAdapter{}, dr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Bid == nil || got.Bid.Payload != "custom" || got.Bid.Price != 1.23 {
		t.Fatalf("unexpected custom bid: %+v", got.Bid)
	}
}
