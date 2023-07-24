package notification_test

import (
	"context"
	"testing"

	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/notification"
	"github.com/bidon-io/bidon-backend/internal/notification/mocks"
	"github.com/bidon-io/bidon-backend/internal/notification/store"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	"github.com/google/go-cmp/cmp"
)

func TestHandler_HandleRound(t *testing.T) {
	ctx := context.Background()
	imp := &schema.Imp{ID: "imp-1"}
	responses := []*adapters.DemandResponse{
		{Bid: &adapters.BidDemandResponse{ID: "bid-1", ImpID: "imp-1", Price: 1.23}},
		{Bid: &adapters.BidDemandResponse{ID: "bid-2", ImpID: "imp-1", Price: 4.56}},
		{Bid: &adapters.BidDemandResponse{ID: "bid-3", ImpID: "imp-1", Price: 7.89}},
		{Bid: &adapters.BidDemandResponse{ID: "bid-4", ImpID: "imp-1", Price: 0.12}},
	}
	expectedBids := []store.Bid{
		{ID: "bid-1", ImpID: "imp-1", Price: 1.23},
		{ID: "bid-2", ImpID: "imp-1", Price: 4.56},
		{ID: "bid-3", ImpID: "imp-1", Price: 7.89},
		{ID: "bid-4", ImpID: "imp-1", Price: 0.12},
	}
	mockRepo := &mocks.AuctionResultRepoMock{}
	mockRepo.CreateOrUpdateFunc = func(ctx context.Context, imp *schema.Imp, bids []store.Bid) error {
		if diff := cmp.Diff(expectedBids, bids); diff != "" {
			t.Errorf("CreateOrUpdate() mismatched arguments (-want, +got)\n%s", diff)
		}
		return nil
	}

	handler := notification.Handler{AuctionResultRepo: mockRepo}

	err := handler.HandleRound(ctx, imp, responses)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandler_HandleStats(t *testing.T) {
	ctx := context.Background()
	imp := &schema.Imp{ID: "imp-1"}
	responses := []*adapters.DemandResponse{
		{Bid: &adapters.BidDemandResponse{ID: "bid-1", ImpID: "imp-1", Price: 1.23}},
		{Bid: &adapters.BidDemandResponse{ID: "bid-2", ImpID: "imp-1", Price: 4.56}},
		{Bid: &adapters.BidDemandResponse{ID: "bid-3", ImpID: "imp-2", Price: 7.89}},
		{Bid: &adapters.BidDemandResponse{ID: "bid-4", ImpID: "imp-1", Price: 0.12}},
	}
	handler := notification.Handler{}

	err := handler.HandleStats(ctx, imp, responses)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandler_HandleShow(t *testing.T) {
	adapters := []*adapters.DemandResponse{
		{Bid: &adapters.BidDemandResponse{ID: "bid-1", ImpID: "imp-1", Price: 1.23, BURL: "http://example.com/burl"}},
		{Bid: &adapters.BidDemandResponse{ID: "bid-2", ImpID: "imp-1", Price: 4.56, BURL: "http://example.com/burl"}},
		{Bid: &adapters.BidDemandResponse{ID: "bid-3", ImpID: "imp-2", Price: 7.89, BURL: "http://example.com/burl"}},
		{Bid: &adapters.BidDemandResponse{ID: "bid-4", ImpID: "imp-1", Price: 0.12, BURL: "http://example.com/burl"}},
	}
	handler := notification.Handler{}

	err := handler.HandleShow(adapters)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandler_HandleWin(t *testing.T) {
	adapters := []*adapters.DemandResponse{
		{Bid: &adapters.BidDemandResponse{ID: "bid-1", ImpID: "imp-1", Price: 1.23, NURL: "http://example.com/win"}},
		{Bid: &adapters.BidDemandResponse{ID: "bid-2", ImpID: "imp-1", Price: 4.56, NURL: "http://example.com/win"}},
		{Bid: &adapters.BidDemandResponse{ID: "bid-3", ImpID: "imp-2", Price: 7.89, NURL: "http://example.com/win"}},
		{Bid: &adapters.BidDemandResponse{ID: "bid-4", ImpID: "imp-1", Price: 0.12, NURL: "http://example.com/win"}},
	}
	handler := notification.Handler{}

	err := handler.HandleWin(adapters)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandler_HandleLoss(t *testing.T) {
	adapters := []*adapters.DemandResponse{
		{Bid: &adapters.BidDemandResponse{ID: "bid-1", ImpID: "imp-1", Price: 1.23, LURL: "http://example.com/loss"}},
		{Bid: &adapters.BidDemandResponse{ID: "bid-2", ImpID: "imp-1", Price: 4.56, LURL: "http://example.com/loss"}},
		{Bid: &adapters.BidDemandResponse{ID: "bid-3", ImpID: "imp-2", Price: 7.89, LURL: "http://example.com/loss"}},
		{Bid: &adapters.BidDemandResponse{ID: "bid-4", ImpID: "imp-1", Price: 0.12, LURL: "http://example.com/loss"}},
	}
	handler := notification.Handler{}

	err := handler.HandleLoss(adapters)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
