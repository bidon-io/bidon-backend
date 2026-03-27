package notification

import (
	"encoding/json"

	"github.com/bidon-io/bidon-backend/internal/adapter"
)

type AuctionResult struct {
	AuctionID             string                 `json:"auction_id"`
	Bids                  []Bid                  `json:"bids"`
	InsightsNotifications []InsightsNotification `json:"insights_notifications,omitempty"`
}

type InsightsNotification struct {
	InsightProvider string `json:"insights_provider,omitempty"`
	Auction         string `json:"auction"`
	Impression      string `json:"impression"`
	Click           string `json:"click"`
}

type Bid struct {
	ID        string      `json:"id"`
	ImpID     string      `json:"impid"`
	Price     float64     `json:"price"`
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
