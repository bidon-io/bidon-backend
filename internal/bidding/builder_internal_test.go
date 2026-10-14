package bidding

import (
	"testing"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
)

func TestBestSlotResponse(t *testing.T) {
	low := &adapters.DemandResponse{DemandID: adapter.AmazonKey, Bid: &adapters.DemandBid{Price: 0.5}}
	high := &adapters.DemandResponse{DemandID: adapter.AmazonKey, Bid: &adapters.DemandBid{Price: 2}}
	nobid := &adapters.DemandResponse{DemandID: adapter.AmazonKey}

	tests := []struct {
		name    string
		slots   []*adapters.DemandResponse
		want    *adapters.DemandResponse
		wantBid bool
	}{
		{name: "highest bid wins", slots: []*adapters.DemandResponse{low, nobid, high}, want: high, wantBid: true},
		{name: "bid beats earlier no-bid", slots: []*adapters.DemandResponse{nobid, low}, want: low, wantBid: true},
		{name: "all no-bid returns first", slots: []*adapters.DemandResponse{nobid}, want: nobid},
		{name: "no slots synthesizes no-bid", slots: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bestSlotResponse(adapter.AmazonKey, tt.slots, 10, 20)
			if tt.want != nil && got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
			if got.IsBid() != tt.wantBid {
				t.Errorf("IsBid() = %v, want %v", got.IsBid(), tt.wantBid)
			}
			if tt.want == nil && (got.DemandID != adapter.AmazonKey || got.StartTS != 10 || got.SendTS != 20) {
				t.Errorf("synthesized = %+v", got)
			}
		})
	}
}
