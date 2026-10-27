package adapters

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/bidding/rendering"
)

// CustomBidParser is implemented by non-OpenRTB adapters that fully own
// parsing RawResponse into Bid. Type-asserted by ParseDemandResponse.
type CustomBidParser interface {
	ParseBids(*DemandResponse) (*DemandResponse, error)
}

// OpenRTBBidEnricher is implemented by OpenRTB adapters that need extra
// extraction after the default DemandBid mapping (e.g. TaurusX payload
// from BidResponse.ext). Common bid.ext fields (signaldata, rendering) are
// handled by the shared parser.
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
// optionally followed by an OpenRTB enricher. On success, Rendering is
// filled when still unset (custom parsers / paths that skip OpenRTB mapping).
func ParseDemandResponse(adapter BidderInterface, dr *DemandResponse) (*DemandResponse, error) {
	if dr == nil {
		return nil, fmt.Errorf("nil demand response")
	}

	var err error
	if p, ok := adapter.(CustomBidParser); ok {
		dr, err = p.ParseBids(dr)
	} else {
		dr, err = parseOpenRTBBids(adapter, dr)
	}
	if err == nil {
		dr.FillRendering()
	}
	return dr, err
}

// AnnotateCatalogFields copies nbr, crid, and ext.crtype out of an OpenRTB
// body onto the response. Non-JSON bodies are left unchanged. Fields already
// set are kept.
func AnnotateCatalogFields(dr *DemandResponse) {
	if dr == nil || dr.RawResponse == "" {
		return
	}
	var body struct {
		NBR     *int32 `json:"nbr"`
		SeatBid []struct {
			Bid []struct {
				CrID string `json:"crid"`
				Ext  struct {
					CrType string `json:"crtype"`
				} `json:"ext"`
			} `json:"bid"`
		} `json:"seatbid"`
	}
	if err := json.Unmarshal([]byte(dr.RawResponse), &body); err != nil {
		return
	}
	if dr.NBR == nil {
		dr.NBR = body.NBR
	}
	if len(body.SeatBid) == 0 || len(body.SeatBid[0].Bid) == 0 {
		return
	}
	bid := body.SeatBid[0].Bid[0]
	if dr.CreativeID == "" {
		dr.CreativeID = bid.CrID
	}
	if dr.CrType == "" {
		dr.CrType = bid.Ext.CrType
	}
}

// FillRendering derives Bid.Rendering from Bid.Ext when unset.
// Used by ParseDemandResponse and non-parse bid paths (Amazon, bid cache).
// OpenRTB parse already sets Rendering via rendering.DecodeBidExt.
func (dr *DemandResponse) FillRendering() {
	if dr == nil || !dr.IsBid() || dr.Bid.Rendering != nil {
		return
	}
	demandID := dr.DemandID
	if demandID == "" {
		demandID = dr.Bid.DemandID
	}
	dr.Bid.Rendering = rendering.ParseFromBidExt(dr.Bid.Ext, demandID)
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

	signaldata, rend, err := rendering.DecodeBidExt(bid.Ext, dr.DemandID)
	if err != nil {
		return dr, err
	}

	dr.Bid = &DemandBid{
		ID:         bid.ID,
		ImpID:      bid.ImpID,
		Price:      bid.Price,
		Payload:    bid.AdM,
		Signaldata: signaldata,
		DemandID:   dr.DemandID,
		AdID:       bid.AdID,
		SeatID:     seat.Seat,
		LURL:       bid.LURL,
		NURL:       bid.NURL,
		BURL:       bid.BURL,
		Ext:        bid.Ext,
		Rendering:  rend,
	}

	if e, ok := adapter.(OpenRTBBidEnricher); ok {
		if err := e.EnrichOpenRTBBid(dr, &bidResponse, seat, bid); err != nil {
			return dr, err
		}
	}

	return dr, nil
}
