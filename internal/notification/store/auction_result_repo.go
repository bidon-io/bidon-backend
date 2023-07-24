package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	"github.com/redis/go-redis/v9"
)

type AuctionResultRepo struct {
	Redis *redis.Client
}

type AuctionResult struct {
	AuctionID string  `json:"auction_id"`
	Rounds    []Round `json:"rounds"`
}

func (a AuctionResult) MarshalBinary() ([]byte, error) {
	return json.Marshal(a)
}

func (a *AuctionResult) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, a)
}

type Round struct {
	RoundID  string  `json:"round_id"`
	Bids     []Bid   `json:"bids"`
	BidFloor float64 `json:"bidfloor"`
}

type Bid struct {
	ID       string      `json:"id"`
	ImpID    string      `json:"impid"`
	Price    float64     `json:"price"`
	Payload  string      `json:"payload"`
	DemandID adapter.Key `json:"demand_id"`
	AdID     string      `json:"adid"`
	SeatID   string      `json:"seatid"`
	LURL     string      `json:"lurl"`
	NURL     string      `json:"nurl"`
	BURL     string      `json:"burl"`
}

func (r AuctionResultRepo) CreateOrUpdate(ctx context.Context, imp *schema.Imp, bids []Bid) error {
	auctionResult, err := r.Find(ctx, imp.AuctionID)
	if err != nil {
		return err
	}

	round := Round{
		RoundID:  imp.RoundID,
		Bids:     bids,
		BidFloor: imp.BidFloor,
	}

	if auctionResult != nil {
		for _, existingRound := range auctionResult.Rounds {
			if existingRound.RoundID == imp.RoundID {
				return fmt.Errorf("round %s already exists", imp.RoundID)
			}
		}
		// This is can be potentially a problem place if we have 2 concurrent requests. Lock should be added
		auctionResult.Rounds = append(auctionResult.Rounds, round)
	} else {
		auctionResult = &AuctionResult{
			AuctionID: imp.AuctionID,
			Rounds:    []Round{round},
		}
	}

	err = auctionResult.Save(ctx, r.Redis)
	if err != nil {
		return err
	}

	return nil
}

func (r AuctionResultRepo) FinalizeResult(ctx context.Context, statsRequest *schema.Stats) error {
	if !statsRequest.Result.IsSuccess() {
		return nil
	}

	winningPrice := statsRequest.Result.ECPM
	fmt.Println(winningPrice)
	auctionResult, err := r.Find(ctx, statsRequest.AuctionID)
	if err != nil {
		return err
	}
	fmt.Println(auctionResult)

	return nil
}

func (r AuctionResultRepo) Find(ctx context.Context, auctionID string) (*AuctionResult, error) {
	auctionResult := &AuctionResult{}
	err := r.Redis.Get(ctx, auctionID).Scan(auctionResult)
	switch err {
	case redis.Nil: // Key does not exist
		return nil, nil
	case nil:
		return auctionResult, nil
	default:
		return nil, err
	}
}

var TTL time.Duration = 24 * time.Hour

func (a *AuctionResult) Save(ctx context.Context, rdb *redis.Client) error {
	err := rdb.Set(ctx, a.AuctionID, a, TTL).Err()
	if err != nil {
		return err
	}

	return nil
}

func (a *AuctionResult) WinningBid() float64 {
	max := 0.0
	for _, round := range a.Rounds {
		for _, bid := range round.Bids {
			if bid.Price > max {
				max = bid.Price
			}
		}
	}
	return max
}
