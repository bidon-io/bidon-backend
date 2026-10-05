package telemetry

import (
	"testing"
	"time"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
)

func TestFloorRuleIsConsistentAcrossEvents(t *testing.T) {
	tests := []struct {
		name         string
		price        float64
		wantRejected bool
	}{
		{name: "at floor is rejected", price: 1.0, wantRejected: true},
		{name: "below floor is rejected", price: 0.5, wantRejected: true},
		{name: "inside old epsilon gap counts", price: 1.0000005, wantRejected: false},
		{name: "above floor counts", price: 1.5, wantRejected: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &MemoryEngine{}
			ev := New(engine, nil).Event()
			params := testAuctionParams()
			params.Request.AdObject.PriceFloor = 1.0
			bid := adapters.DemandResponse{
				DemandID: adapter.BidmachineKey,
				Status:   200,
				Bid:      &adapters.DemandBid{Price: tt.price},
				SendTS:   1,
			}

			ev.DSPResponseReceived(params, &bid)
			ev.AuctionCompleted(AuctionCompletedParams{Params: params, Bids: []adapters.DemandResponse{bid}, Started: time.Now()})

			var rejected bool
			var completed Record
			for _, rec := range engine.Records() {
				switch rec.EventName {
				case EventDSPResponseRejected:
					rejected = true
					if rec.PriceFloor != 1.0 {
						t.Errorf("rejected price_floor = %v, want 1.0", rec.PriceFloor)
					}
				case EventAuctionCompleted:
					completed = rec
				}
			}
			if rejected != tt.wantRejected {
				t.Errorf("rejected = %v, want %v", rejected, tt.wantRejected)
			}
			if won := completed.WinnerDSP != ""; won == rejected {
				t.Errorf("winner_dsp = %q while rejected = %v; a bid must be exactly one of the two", completed.WinnerDSP, rejected)
			}
			if completed.PriceFloor != 1.0 {
				t.Errorf("completed price_floor = %v, want 1.0", completed.PriceFloor)
			}
		})
	}
}

func TestParticipantCount(t *testing.T) {
	bids := []adapters.DemandResponse{
		{DemandID: adapter.BidmachineKey, SendTS: 1},
		{DemandID: adapter.AmazonKey, SendTS: 1},
		{DemandID: adapter.AmazonKey, SendTS: 1},
		{DemandID: adapter.MetaKey},
	}
	if got := participantCount(bids); got != 2 {
		t.Errorf("participantCount = %d, want 2 (Amazon slots count once, unsent excluded)", got)
	}
}
