package adapters

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/prebid/openrtb/v19/openrtb2"
)

// CustomBidParser is implemented by non-OpenRTB adapters that fully own
// parsing RawResponse into Bid. Type-asserted by ParseDemandResponse.
type CustomBidParser interface {
	ParseBids(*DemandResponse) (*DemandResponse, error)
}

// OpenRTBBidEnricher is implemented by OpenRTB adapters that need extra
// extraction after the default NormalizedBid mapping.
type OpenRTBBidEnricher interface {
	EnrichOpenRTBBid(
		dr *DemandResponse,
		bidResp *openrtb2.BidResponse,
		seat openrtb2.SeatBid,
		bid openrtb2.Bid,
	) error
}

// ParseDemandResponse unpacks a demand HTTP response into a bid DTO.
// Custom parsers take precedence; otherwise the shared OpenRTB path runs,
// optionally followed by an OpenRTB enricher.
func ParseDemandResponse(adapter BidderInterface, dr *DemandResponse) (*DemandResponse, error) {
	if dr == nil {
		return nil, fmt.Errorf("nil demand response")
	}

	if p, ok := adapter.(CustomBidParser); ok {
		return p.ParseBids(dr)
	}

	return parseOpenRTBBids(adapter, dr)
}

func parseOpenRTBBids(adapter BidderInterface, dr *DemandResponse) (*DemandResponse, error) {
	switch dr.Status {
	case http.StatusNoContent:
		return dr, nil
	case http.StatusServiceUnavailable, http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden:
		return dr, fmt.Errorf("unauthorized request: %s", strconv.Itoa(dr.Status))
	case http.StatusOK:
		// proceed
	default:
		return dr, fmt.Errorf("unexpected status code: %s", strconv.Itoa(dr.Status))
	}

	var bidResponse openrtb2.BidResponse
	if err := json.Unmarshal([]byte(dr.RawResponse), &bidResponse); err != nil {
		return dr, err
	}

	if len(bidResponse.SeatBid) == 0 || len(bidResponse.SeatBid[0].Bid) == 0 {
		return dr, nil
	}

	seat := bidResponse.SeatBid[0]
	bid := seat.Bid[0]

	dr.Bid = &NormalizedBid{
		ID:       bid.ID,
		ImpID:    bid.ImpID,
		Price:    bid.Price,
		Payload:  bid.AdM,
		DemandID: dr.DemandID,
		AdID:     bid.AdID,
		SeatID:   seat.Seat,
		LURL:     bid.LURL,
		NURL:     bid.NURL,
		BURL:     bid.BURL,
		Ext:      bid.Ext,
	}

	if e, ok := adapter.(OpenRTBBidEnricher); ok {
		if err := e.EnrichOpenRTBBid(dr, &bidResponse, seat, bid); err != nil {
			return dr, err
		}
	}

	return dr, nil
}
