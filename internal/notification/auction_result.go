package notification

import (
	"encoding/json"

	"github.com/shopspring/decimal"

	"github.com/bidon-io/bidon-backend/internal/adapter"
)

type AuctionResult struct {
	AuctionID string `json:"auction_id"`
	Bids      []Bid  `json:"bids"`
}

type Bid struct {
	ID        string      `json:"id"`
	ImpID     string      `json:"impid"`
	Price     decimal.Decimal     `json:"price"`
	DemandID  adapter.Key `json:"demand_id"`
	AdID      string      `json:"adid"`
	SeatID    string      `json:"seatid"`
	LURL      string      `json:"lurl"`
	NURL      string      `json:"nurl"`
	BURL      string      `json:"burl"`
	RequestID string      `json:"request_id"`
}

func (a *AuctionResult) MarshalBinary() ([]byte, error) {
	return json.Marshal(a)
}

func (a *AuctionResult) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, a)
}
