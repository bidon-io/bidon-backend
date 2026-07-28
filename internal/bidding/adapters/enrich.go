package adapters

import "github.com/bidon-io/bidon-backend/internal/bidding/rendering"

// EnrichBid applies post-parse enrichment steps to a demand bid.
// Adapters only map network fields (including raw Bid.Ext); EnrichBid is the
// single place that derives Bidon-facing bid attributes. Add new steps here
// rather than in individual adapters.
func EnrichBid(dr *DemandResponse) {
	if dr == nil || !dr.IsBid() {
		return
	}

	enrichRendering(dr)
}

// enrichRendering fills Bid.Rendering from Bid.Ext when Rendering is unset.
// Cached bids already carry a parsed Rendering (policy A) and are left alone.
func enrichRendering(dr *DemandResponse) {
	if dr.Bid.Rendering != nil {
		return
	}
	demandID := dr.DemandID
	if demandID == "" {
		demandID = dr.Bid.DemandID
	}
	dr.Bid.Rendering = rendering.ParseFromBidExt(dr.Bid.Ext, demandID)
}
