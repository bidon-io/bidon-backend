package dspsimulator

import (
	"github.com/prebid/openrtb/v19/openrtb2"
)

type Service struct {
	store *ResponseStore
}

func NewService(store *ResponseStore) *Service {
	return &Service{store: store}
}

func (s *Service) HandleBidRequest(req *openrtb2.BidRequest) *openrtb2.BidResponse {
	keys := BuildKeys(req)

	var seatbids []openrtb2.SeatBid
	currencies := make(map[string]bool)
	matched := false

	for _, key := range keys {
		resp := s.store.Lookup(key)
		if resp == nil {
			continue
		}

		matched = true

		clone := cloneResponse(resp)
		clone.ID = req.ID
		for si := range clone.SeatBid {
			for bi := range clone.SeatBid[si].Bid {
				clone.SeatBid[si].Bid[bi].ImpID = req.Imp[0].ID
			}
		}

		seatbids = append(seatbids, clone.SeatBid...)
		if resp.Cur != "" {
			currencies[resp.Cur] = true
		}
	}

	if !matched {
		return nil
	}

	cur := ""
	for c := range currencies {
		cur = c
	}

	return &openrtb2.BidResponse{
		ID:      req.ID,
		SeatBid: seatbids,
		Cur:     cur,
	}
}

func cloneResponse(resp *openrtb2.BidResponse) *openrtb2.BidResponse {
	clone := *resp
	clone.SeatBid = make([]openrtb2.SeatBid, len(resp.SeatBid))
	for i, sb := range resp.SeatBid {
		clone.SeatBid[i] = sb
		clone.SeatBid[i].Bid = make([]openrtb2.Bid, len(sb.Bid))
		copy(clone.SeatBid[i].Bid, sb.Bid)
	}
	return &clone
}
