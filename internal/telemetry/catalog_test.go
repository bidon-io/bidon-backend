package telemetry

import (
	"testing"
	"time"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	telemetryv1 "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/telemetry/v1"
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

			rejectedEvents := EventsOf[*telemetryv1.DspResponseRejected](engine)
			rejected := len(rejectedEvents) > 0
			if rejected && rejectedEvents[0].GetPriceFloor() != 1.0 {
				t.Errorf("rejected price_floor = %v, want 1.0", rejectedEvents[0].GetPriceFloor())
			}
			if rejected != tt.wantRejected {
				t.Errorf("rejected = %v, want %v", rejected, tt.wantRejected)
			}
			completedEvents := EventsOf[*telemetryv1.AuctionCompleted](engine)
			if len(completedEvents) != 1 {
				t.Fatalf("auction_completed events = %d, want 1", len(completedEvents))
			}
			completed := completedEvents[0]
			if won := completed.GetWinnerDsp() != ""; won == rejected {
				t.Errorf("winner_dsp = %q while rejected = %v; a bid must be exactly one of the two", completed.GetWinnerDsp(), rejected)
			}
			if completed.GetPriceFloor() != 1.0 {
				t.Errorf("completed price_floor = %v, want 1.0", completed.GetPriceFloor())
			}
		})
	}
}

func TestDSPResponseReceivedCopiesCreativeFields(t *testing.T) {
	engine := &MemoryEngine{}
	ev := New(engine, nil).Event()
	dr := &adapters.DemandResponse{
		DemandID:    adapter.BidmachineKey,
		Status:      200,
		RawResponse: `{"nbr":2,"seatbid":[{"bid":[{"crid":"cr-9","price":1.2,"ext":{"crtype":"MRAID"}}]}]}`,
		Bid:         &adapters.DemandBid{Price: 1.2},
		SendTS:      1,
		EndTS:       5,
	}
	ev.DSPResponseReceived(testAuctionParams(), dr)

	got := EventsOf[*telemetryv1.DspResponseReceived](engine)
	if len(got) != 1 {
		t.Fatalf("dsp_response_received = %d, want 1", len(got))
	}
	if got[0].GetCreativeId() != "cr-9" || got[0].GetCrtype() != "MRAID" || got[0].GetNbr() != 2 {
		t.Errorf("creative fields = crid %q crtype %q nbr %d", got[0].GetCreativeId(), got[0].GetCrtype(), got[0].GetNbr())
	}
}

func TestAdImpressionKeepsClientEventID(t *testing.T) {
	engine := &MemoryEngine{}
	ev := New(engine, nil).Event()
	client := &telemetryv1.AdImpression{
		Envelope:         &telemetryv1.Envelope{EventId: "client-1", AuctionId: "auc", AppId: 7},
		RenderDurationMs: 12,
		OmEnabled:        true,
		Dsp:              "dsp",
	}
	ev.AdImpression(Identity{AppID: 9, AuctionID: "other"}, "dsp", 1.5, client)
	ev.AdImpression(Identity{AppID: 9, AuctionID: "auc-synth", SessionID: "sess"}, "dsp", 1.5, nil)

	got := EventsOf[*telemetryv1.AdImpression](engine)
	if len(got) != 2 {
		t.Fatalf("ad_impression = %d, want 2", len(got))
	}
	if got[0].GetEnvelope().GetEventId() != "client-1" || got[0].GetRenderDurationMs() != 12 || !got[0].GetOmEnabled() {
		t.Errorf("client row was rewritten: %+v", got[0])
	}
	if got[1].GetEnvelope().GetEventId() == "" || got[1].GetEnvelope().GetEventId() == "client-1" {
		t.Errorf("synthesized event_id = %q", got[1].GetEnvelope().GetEventId())
	}
	if got[1].GetPrice() != 1.5 || got[1].GetEnvelope().GetAuctionId() != "auc-synth" {
		t.Errorf("synthesized row = %+v", got[1])
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
