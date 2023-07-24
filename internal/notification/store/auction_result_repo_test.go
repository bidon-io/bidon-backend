package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/bidon-io/bidon-backend/internal/notification/store"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	"github.com/go-redis/redismock/v9"
	"github.com/google/go-cmp/cmp"
)

func TestAuctionResultRepo_CreateOrUpdate(t *testing.T) {
	ctx := context.Background()
	imp := &schema.Imp{
		AuctionID: "auction-1",
		RoundID:   "round-1",
		BidFloor:  0.5,
	}
	bids := []store.Bid{
		{ID: "bid-1", ImpID: "imp-1", Price: 1.23},
		{ID: "bid-2", ImpID: "imp-1", Price: 4.56},
		{ID: "bid-3", ImpID: "imp-2", Price: 7.89},
		{ID: "bid-4", ImpID: "imp-1", Price: 0.12},
	}
	expectedAuctionResult := &store.AuctionResult{
		AuctionID: "auction-1",
		Rounds: []store.Round{
			{
				RoundID:  "round-1",
				Bids:     bids,
				BidFloor: 0.5,
			},
		},
	}
	rdb, mock := redismock.NewClientMock()
	mock.ExpectGet("auction-1").RedisNil()
	mock.ExpectSet("auction-1", expectedAuctionResult, 24*time.Hour).SetVal("OK")

	repo := store.AuctionResultRepo{Redis: rdb}
	err := repo.CreateOrUpdate(ctx, imp, bids)

	if mock.ExpectationsWereMet() != nil {
		t.Errorf("expectation not met: %v", mock.ExpectationsWereMet())
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAuctionResultRepo_CreateOrUpdate_DuplicateRound(t *testing.T) {
	ctx := context.Background()
	imp := &schema.Imp{
		AuctionID: "auction-1",
		RoundID:   "round-1",
		BidFloor:  0.5,
	}
	bids := []store.Bid{
		{ID: "bid-1", ImpID: "imp-1", Price: 1.23},
		{ID: "bid-2", ImpID: "imp-1", Price: 4.56},
		{ID: "bid-3", ImpID: "imp-2", Price: 7.89},
		{ID: "bid-4", ImpID: "imp-1", Price: 0.12},
	}
	existingAuctionResult := &store.AuctionResult{
		AuctionID: "auction-1",
		Rounds: []store.Round{
			{
				RoundID:  "round-1",
				Bids:     bids,
				BidFloor: 0.5,
			},
		},
	}
	bytes, _ := existingAuctionResult.MarshalBinary()
	rdb, mock := redismock.NewClientMock()
	mock.ExpectGet("auction-1").SetVal(string(bytes))

	repo := store.AuctionResultRepo{Redis: rdb}
	err := repo.CreateOrUpdate(ctx, imp, bids)

	if err.Error() != "round round-1 already exists" {
		t.Errorf("expectation not met: %v", err)
	}
	if err == nil {
		t.Errorf("expected error, got not errors")
	}
}

func TestAuctionResultRepo_Find(t *testing.T) {
	ctx := context.Background()
	expectedAuctionResult := &store.AuctionResult{
		AuctionID: "auction-1",
		Rounds: []store.Round{
			{
				RoundID:  "round-1",
				Bids:     []store.Bid{},
				BidFloor: 0.5,
			},
		},
	}
	bytes, _ := expectedAuctionResult.MarshalBinary()
	rdb, mock := redismock.NewClientMock()
	mock.ExpectGet("auction-1").SetVal(string(bytes))

	repo := store.AuctionResultRepo{Redis: rdb}
	actualAuctionResult, err := repo.Find(ctx, "auction-1")

	if mock.ExpectationsWereMet() != nil {
		t.Errorf("expectation not met: %v", mock.ExpectationsWereMet())
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if diff := cmp.Diff(expectedAuctionResult, actualAuctionResult); diff != "" {
		t.Errorf("expectedAuctionResult -> %+v mismatch \n(-want, +got)\n%s", expectedAuctionResult, diff)
	}
}

func TestAuctionResultRepo_Find_NotFound(t *testing.T) {
	ctx := context.Background()
	rdb, mock := redismock.NewClientMock()
	mock.ExpectGet("auction-1").RedisNil()

	repo := store.AuctionResultRepo{Redis: rdb}
	actualAuctionResult, err := repo.Find(ctx, "auction-1")

	if mock.ExpectationsWereMet() != nil {
		t.Errorf("expectation not met: %v", mock.ExpectationsWereMet())
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if actualAuctionResult != nil {
		t.Errorf("actualAuctionResult not nil")
	}
}

func TestAuctionResult_Save(t *testing.T) {
	ctx := context.Background()
	auctionResult := &store.AuctionResult{
		AuctionID: "auction-1",
		Rounds: []store.Round{
			{
				RoundID:  "round-1",
				Bids:     []store.Bid{},
				BidFloor: 0.5,
			},
		},
	}
	rdb, mock := redismock.NewClientMock()
	mock.ExpectSet("auction-1", auctionResult, 24*time.Hour).SetVal("OK")

	err := auctionResult.Save(ctx, rdb)

	if mock.ExpectationsWereMet() != nil {
		t.Errorf("expectation not met: %v", mock.ExpectationsWereMet())
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAuctionResult_WinningBid(t *testing.T) {
	auctionResult := &store.AuctionResult{
		AuctionID: "auction-1",
		Rounds: []store.Round{
			{
				RoundID: "round-1",
				Bids: []store.Bid{
					{ID: "bid-1", ImpID: "imp-1", Price: 1.23},
					{ID: "bid-2", ImpID: "imp-1", Price: 4.56},
					{ID: "bid-3", ImpID: "imp-2", Price: 7.89},
					{ID: "bid-4", ImpID: "imp-1", Price: 0.12},
				},
				BidFloor: 0.5,
			},
			{
				RoundID: "round-2",
				Bids: []store.Bid{
					{ID: "bid-5", ImpID: "imp-1", Price: 2.34},
					{ID: "bid-6", ImpID: "imp-1", Price: 5.67},
					{ID: "bid-7", ImpID: "imp-2", Price: 8.9},
					{ID: "bid-8", ImpID: "imp-1", Price: 0.23},
				},
				BidFloor: 0.5,
			},
		},
	}

	winningBid := auctionResult.WinningBid()

	if winningBid != 8.9 {
		t.Errorf("expected winningBid 8.9, got %f", winningBid)
	}
}
