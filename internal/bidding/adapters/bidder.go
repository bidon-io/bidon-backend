package adapters

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

type BidderInterface interface {
	// CreateRequest makes the HTTP requests which should be made to fetch bids.
	CreateRequest(openrtb.BidRequest, *schema.AuctionRequest) (openrtb.BidRequest, error)

	// ExecuteRequest sends request to bidder endpoint.
	// It must only populate Status / RawResponse (and transport errors);
	// bid parsing is done by ParseDemandResponse at the builder call site.
	ExecuteRequest(context.Context, *http.Client, openrtb.BidRequest) *DemandResponse
}

type Bidder struct {
	Adapter BidderInterface
	Client  *http.Client
}

type Builder func(adapter.ProcessedConfigsMap, *http.Client) (*Bidder, error)

type DemandResponse struct {
	DemandID    adapter.Key
	RequestID   string
	RawRequest  string
	RawResponse string
	Status      int
	Bid         *NormalizedBid
	Error       error
	ImpID       string
	TagID       string
	PlacementID string
	SlotUUID    string
	TimeoutURL  string
	StartTS     int64
	EndTS       int64
	Token       Token
}

func (dr *DemandResponse) IsBid() bool {
	return dr.Bid != nil
}

func (dr *DemandResponse) ErrorMessage() string {
	errMsg := ""
	if dr.Error != nil {
		errMsg = dr.Error.Error()
	}

	if dr.Token.Status != TokenStatusSuccess {
		if errMsg != "" {
			errMsg += "; "
		}
		errMsg += "Token Status: " + dr.Token.Status
	}

	return errMsg
}

func (dr *DemandResponse) Price() float64 {
	price := float64(0)
	if dr.IsBid() {
		price = dr.Bid.Price
	}
	return price
}

// CanCache returns true if the bidder can cache the bid response. For now, it returns false just for BM and Amazon.
// Probably should be part of the BidderInterface in the future.
func (dr *DemandResponse) CanCache() bool {
	if !dr.IsBid() {
		return false
	}
	if dr.DemandID == adapter.BidmachineKey || dr.DemandID == adapter.AmazonKey {
		return false
	}

	return true
}

// NormalizedBid is the DSP-agnostic bid after network response parsing.
// OpenRTB adapters fill Ext from seatbid.bid.ext; proprietary adapters leave it nil.
type NormalizedBid struct {
	Payload string
	// Signaldata is extracted from seatbid.bid.ext.signaldata by the shared OpenRTB parser.
	Signaldata string
	ID         string
	ImpID      string
	AdID       string
	SeatID     string
	DemandID   adapter.Key
	Price      float64
	LURL       string
	NURL       string
	BURL       string
	// Ext is the raw OpenRTB seatbid.bid.ext (nil for non-OpenRTB demands).
	Ext json.RawMessage
}

type Token struct {
	Value   string
	Status  string
	StartTS int64
	EndTS   int64
}

const (
	TokenStatusSuccess = "SUCCESS"
)
