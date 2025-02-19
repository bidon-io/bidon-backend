package bidding_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/go-cmp/cmp"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	"github.com/bidon-io/bidon-backend/pkg/clock"
)

func TestBidCache_ApplyBidCache(t *testing.T) {
	redisClient, mock := redismock.NewClusterMock()
	mockTime := clock.NewMock()
	mockTime.Set(time.Now())
	bidCache := &bidding.BidCache{Redis: redisClient, Clock: mockTime}

	ctx := context.Background()
	br := &schema.BiddingRequest{
		BaseRequest: schema.BaseRequest{
			Session: schema.Session{ID: "session1"},
			Ext:     "{\"bid_cache\": true}",
		},
		AdType: "banner",
	}
	br.NormalizeValues()

	tests := []struct {
		name     string
		bids     []adapters.DemandResponse
		cacheGet bidding.Cache
		cacheSet bidding.Cache
		want     []adapters.DemandResponse
	}{
		{
			name:     "no cache, no bids",
			bids:     []adapters.DemandResponse{},
			cacheGet: bidding.Cache{},
			cacheSet: bidding.Cache{},
			want:     []adapters.DemandResponse{},
		},
		{
			name: "no cache, has bids",
			bids: []adapters.DemandResponse{
				{DemandID: adapter.BidmachineKey, Bid: &adapters.BidDemandResponse{Price: 1.0}},
				{DemandID: adapter.ApplovinKey, Bid: &adapters.BidDemandResponse{Price: 2.0}},
				{DemandID: adapter.MetaKey, Bid: &adapters.BidDemandResponse{Price: 3.0}},
			},
			cacheGet: bidding.Cache{},
			cacheSet: bidding.Cache{
				Bids: map[adapter.Key]bidding.CacheEntry{
					adapter.ApplovinKey: {
						Bid:       adapters.DemandResponse{DemandID: adapter.ApplovinKey, Bid: &adapters.BidDemandResponse{Price: 2.0}},
						CreatedAt: mockTime.Now(),
						AuctionID: "session1",
					},
				},
			},
			want: []adapters.DemandResponse{
				{DemandID: adapter.BidmachineKey, Bid: &adapters.BidDemandResponse{Price: 1.0}},
				{DemandID: adapter.MetaKey, Bid: &adapters.BidDemandResponse{Price: 3.0}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aucResult := &bidding.AuctionResult{
				Bids: tt.bids,
			}
			bytes, _ := tt.cacheGet.MarshalBinary()
			if len(tt.cacheGet.Bids) > 0 {
				mock.ExpectGetDel("bidding:session1:banner").SetVal(string(bytes))
			} else {
				mock.ExpectGetDel("bidding:session1:banner").RedisNil()
			}
			if len(tt.cacheSet.Bids) > 0 {
				bytes, _ := tt.cacheSet.MarshalBinary()
				mock.ExpectSet("bidding:session1:banner", string(bytes), bidding.TTL).SetVal("OK")
			}

			got := bidCache.ApplyBidCache(ctx, br, aucResult)

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Create() mismatch (-want +got):\n%s", diff)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet redis expectations: %+v", err)
			}
		})
	}

	// path: cache is empty, result has bids
	//mock.ExpectGetDel("bidding:session1:banner").SetVal("")
	//mock.ExpectSet("bidding:session1:banner", "", bidding.TTL).SetVal("OK")
	//
	//response := bidCache.ApplyBidCache(ctx, br, aucResult)
	//if len(response) != 1 {
	//	t.Errorf("expected 1 bid, got %d", len(response))
	//}
	//if response[0].DemandID != adapter.ApplovinKey {
	//	t.Errorf("expected demand ID %s, got %s", adapter.ApplovinKey, response[0].DemandID)
	//}

	//// Edge case : cache has expired entries
	//expiredCache := bidding.Cache{
	//	Bids: map[adapter.Key]bidding.CacheEntry{
	//		adapter.BidmachineKey: {Bid: adapters.DemandResponse{DemandID: adapter.BidmachineKey, Bid: &adapters.BidDemandResponse{Price: 1.0}}, CreatedAt: time.Now().Add(-10 * time.Minute)},
	//	},
	//}
	//bytes, _ := expiredCache.MarshalBinary()
	//mock.ExpectGetDel("bidding:session1:banner").SetVal(string(bytes))
	//mock.ExpectSet("bidding:session1:banner", "", bidding.TTL).SetVal("OK")
	//
	//response = bidCache.ApplyBidCache(ctx, br, aucResult)
	//if len(response) != 1 {
	//	t.Errorf("expected 1 bid, got %d", len(response))
	//}
	//if response[0].DemandID != adapter.ApplovinKey {
	//	t.Errorf("expected demand ID %s, got %s", adapter.ApplovinKey, response[0].DemandID)
	//}
	//
	//// Edge case: no bids in result and cache
	//aucResult.Bids = []adapters.DemandResponse{}
	//mock.ExpectGetDel("bidding:session1:banner").SetVal("")
	//mock.ExpectSet("bidding:session1:banner", "", bidding.TTL).SetVal("OK")
	//
	//response = bidCache.ApplyBidCache(ctx, br, aucResult)
	//if len(response) != 0 {
	//	t.Errorf("expected 0 bids, got %d", len(response))
	//}
	//
	//// Edge case: no bid_cache field in request
	//br.Ext = ""
	//response = bidCache.ApplyBidCache(ctx, br, aucResult)
	//if len(response) != 0 {
	//	t.Errorf("expected 0 bids, got %d", len(response))
	//}
	//
	//if err := mock.ExpectationsWereMet(); err != nil {
	//	t.Errorf("unmet expectations: %v", err)
	//}
	// expectation: '{"Bids":{"applovin":{"Bid":{"DemandID":"applovin","RequestID":"","RawRequest":"","RawResponse":"","Status":0,"Bid":{"Payload":"","Signaldata":"","ID":"","ImpID":"","AdID":"","SeatID":"","DemandID":"","Price":2,"LURL":"","NURL":"","BURL":""},"Error":null,"TagID":"","PlacementID":"","SlotUUID":"","TimeoutURL":"","StartTS":0,"EndTS":0,"Token":{"Value":"","Status":"","StartTS":0,"EndTS":0}},"CreatedAt":"0001-01-01T00:00:00Z","AuctionID":""}}}',
	//    but gave: '&{Bids:map[applovin:{Bid:{DemandID:applovin RequestID: RawRequest: RawResponse: Status:0 Bid:0x140000ca370 Error:<nil> TagID: PlacementID: SlotUUID: TimeoutURL: StartTS:0 EndTS:0 Token:{Value: Status: StartTS:0 EndTS:0}} CreatedAt:2025-02-19 11:12:11.249578 +0100 CET m=+0.005820168 AuctionID:session1}]}'
	// expectation: '{"Bids":{"applovin":{"Bid":{"DemandID":"applovin","RequestID":"","RawRequest":"","RawResponse":"","Status":0,"Bid":{"Payload":"","Signaldata":"","ID":"","ImpID":"","AdID":"","SeatID":"","DemandID":"","Price":2,"LURL":"","NURL":"","BURL":""},"Error":null,"TagID":"","PlacementID":"","SlotUUID":"","TimeoutURL":"","StartTS":0,"EndTS":0,"Token":{"Value":"","Status":"","StartTS":0,"EndTS":0}},"CreatedAt":"2025-02-19T12:51:20.716083+01:00","AuctionID":"session1"}}}',
	//    but gave: '{"Bids":{"applovin":{"Bid":{"DemandID":"applovin","RequestID":"","RawRequest":"","RawResponse":"","Status":0,"Bid":{"Payload":"","Signaldata":"","ID":"","ImpID":"","AdID":"","SeatID":"","DemandID":"","Price":2,"LURL":"","NURL":"","BURL":""},"Error":null,"TagID":"","PlacementID":"","SlotUUID":"","TimeoutURL":"","StartTS":0,"EndTS":0,"Token":{"Value":"","Status":"","StartTS":0,"EndTS":0}},"CreatedAt":"2025-02-19T12:51:20.714632+01:00","AuctionID":"session1"}}}'

}
