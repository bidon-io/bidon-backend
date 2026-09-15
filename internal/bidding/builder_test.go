package bidding_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/prebid/openrtb/v19/openrtb2"
	"go.uber.org/goleak"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/auction"
	"github.com/bidon-io/bidon-backend/internal/bidding"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters/bidmachine"
	"github.com/bidon-io/bidon-backend/internal/bidding/mocks"
	"github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	"github.com/bidon-io/bidon-backend/internal/telemetry"
)

func testApp(id int64) *sdkapi.App {
	return &sdkapi.App{ID: id}
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestBuilder_Build(t *testing.T) {
	auctionConfig := auction.Config{
		Bidding: []adapter.Key{adapter.BidmachineKey},
	}

	auctionConfigV2 := auction.Config{
		Demands: []adapter.Key{adapter.UnityAdsKey},
		Bidding: []adapter.Key{adapter.BidmachineKey},
		Timeout: 15000,
	}

	adaptersBuilder := &mocks.AdaptersBuilderMock{
		BuildFunc: func(_ adapter.Key, cfg adapter.ProcessedConfigsMap) (*adapters.Bidder, error) {
			adpt := &bidmachine.BidmachineAdapter{
				Endpoint: cfg[adapter.BidmachineKey]["endpoint"].(string),
				SellerID: cfg[adapter.BidmachineKey]["seller_id"].(string),
			}

			bidder := &adapters.Bidder{
				Adapter: adpt,
				Client:  http.DefaultClient,
			}

			return bidder, nil
		},
	}
	defer http.DefaultClient.CloseIdleConnections()

	notificationHandler := &mocks.NotificationHandlerMock{
		HandleBiddingRoundFunc: func(_ context.Context, _ *schema.AdObject, _ bidding.AuctionResult, _ string, _ string) error {
			return nil
		},
	}

	bidCacher := &mocks.BidCacherMock{ // Pass through bids
		ApplyBidCacheFunc: func(_ context.Context, _ *schema.AuctionRequest, aucRes *bidding.AuctionResult) []adapters.DemandResponse {
			return aucRes.Bids
		},
	}

	tests := []struct {
		name                string
		adaptersBuilder     bidding.AdaptersBuilder
		notificationHandler bidding.NotificationHandler
		buildParams         *bidding.BuildParams
		expectedResult      adapters.DemandResponse
		expectedError       error
	}{
		{
			name:                "successful build",
			adaptersBuilder:     adaptersBuilder,
			notificationHandler: notificationHandler,
			buildParams: &bidding.BuildParams{
				App: testApp(1),
				AdapterConfigs: adapter.ProcessedConfigsMap{
					adapter.BidmachineKey: {
						"endpoint":  "https://example.com",
						"seller_id": "1",
					},
				},
				AuctionRequest: schema.AuctionRequest{
					AdObject: schema.AdObject{
						Demands: map[adapter.Key]map[string]any{
							adapter.BidmachineKey: {
								"token": "token",
							},
						},
					},
					Adapters: schema.Adapters{
						adapter.BidmachineKey: {
							Version:    "1.0.0",
							SDKVersion: "1.0.0",
						},
					},
				},
				BiddingAdapters: auctionConfig.Bidding,
			},
			expectedResult: adapters.DemandResponse{
				Status:   204,
				DemandID: adapter.BidmachineKey,
			},
			expectedError: nil,
		},
		{
			name:                "round-less successful build v2",
			adaptersBuilder:     adaptersBuilder,
			notificationHandler: notificationHandler,
			buildParams: &bidding.BuildParams{
				App: testApp(1),
				AdapterConfigs: adapter.ProcessedConfigsMap{
					adapter.BidmachineKey: {
						"endpoint":  "https://example.com",
						"seller_id": "1",
					},
				},
				AuctionRequest: schema.AuctionRequest{
					AdObject: schema.AdObject{
						Demands: map[adapter.Key]map[string]any{
							adapter.BidmachineKey: {
								"token":           "bid_token",
								"status":          "SUCCESS",
								"token_start_ts":  169157953564,
								"token_finish_ts": 169157953564,
							},
						},
					},
					Adapters: schema.Adapters{
						adapter.BidmachineKey: {
							Version:    "1.0.0",
							SDKVersion: "1.0.0",
						},
					},
				},
				BiddingAdapters: auctionConfigV2.Bidding,
			},
			expectedResult: adapters.DemandResponse{
				Status:   204,
				DemandID: adapter.BidmachineKey,
			},
			expectedError: nil,
		},
		{
			name:                "round-less build v2 with no token",
			adaptersBuilder:     adaptersBuilder,
			notificationHandler: notificationHandler,
			buildParams: &bidding.BuildParams{
				App: testApp(1),
				AuctionRequest: schema.AuctionRequest{
					AdObject: schema.AdObject{
						Demands: map[adapter.Key]map[string]any{
							adapter.BidmachineKey: {"status": "SUCCESS"},
						},
					},
					Adapters: schema.Adapters{
						adapter.BidmachineKey: {
							Version:    "1.0.0",
							SDKVersion: "1.0.0",
						},
					},
				},
				BiddingAdapters: auctionConfigV2.Bidding,
			},
			expectedResult: adapters.DemandResponse{
				Status:   204,
				DemandID: adapter.BidmachineKey,
			},
			expectedError: bidding.ErrNoAdaptersMatched,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := &bidding.Builder{
				AdaptersBuilder:     tt.adaptersBuilder,
				NotificationHandler: tt.notificationHandler,
				BidCacher:           bidCacher,
			}

			result, err := builder.HoldAuction(context.Background(), tt.buildParams)

			if err != nil && tt.expectedError == nil {
				t.Errorf("unexpected error: %v", err)
			}

			if err == nil && tt.expectedError != nil {
				t.Errorf("expected error: %v, but got nil", tt.expectedError)
			}

			if len(result.Bids) > 0 && result.Bids[0].Status != tt.expectedResult.Status {
				t.Errorf("expected result: %+v, but got %+v", tt.expectedResult, result)
			}
		})
	}
}

type scriptedAdapter struct {
	response *adapters.DemandResponse
}

func (a scriptedAdapter) BuildImpression(openrtb.BidRequest, *schema.AuctionRequest) (*openrtb2.Imp, adapters.RTBRequestOptions, error) {
	return &openrtb2.Imp{ID: "imp-1"}, adapters.RTBRequestOptions{}, nil
}

func (a scriptedAdapter) ExecuteRequest(context.Context, *http.Client, openrtb.BidRequest) *adapters.DemandResponse {
	return a.response
}

func (a scriptedAdapter) ParseBids(dr *adapters.DemandResponse) (*adapters.DemandResponse, error) {
	return dr, nil
}

func TestBuilder_HoldAuction_EmitsDSPTelemetry(t *testing.T) {
	engine := &telemetry.MemoryEngine{}
	responses := map[adapter.Key]*adapters.DemandResponse{
		adapter.BidmachineKey: {
			DemandID: adapter.BidmachineKey,
			Status:   http.StatusOK,
			Bid:      &adapters.DemandBid{Price: 2.5},
		},
		adapter.MetaKey: {
			DemandID: adapter.MetaKey,
			Status:   http.StatusNoContent,
		},
		adapter.VungleKey: {
			DemandID: adapter.VungleKey,
			Error:    context.DeadlineExceeded,
		},
	}
	keys := []adapter.Key{adapter.BidmachineKey, adapter.MetaKey, adapter.VungleKey}

	builder := &bidding.Builder{
		AdaptersBuilder: &mocks.AdaptersBuilderMock{
			BuildFunc: func(key adapter.Key, _ adapter.ProcessedConfigsMap) (*adapters.Bidder, error) {
				return &adapters.Bidder{Adapter: scriptedAdapter{response: responses[key]}, Client: http.DefaultClient}, nil
			},
		},
		NotificationHandler: &mocks.NotificationHandlerMock{
			HandleBiddingRoundFunc: func(context.Context, *schema.AdObject, bidding.AuctionResult, string, string) error {
				return nil
			},
		},
		BidCacher: &mocks.BidCacherMock{
			ApplyBidCacheFunc: func(_ context.Context, _ *schema.AuctionRequest, aucRes *bidding.AuctionResult) []adapters.DemandResponse {
				return aucRes.Bids
			},
		},
		Telemetry: &telemetry.Logger{Engine: engine},
	}

	adaptersCfg := schema.Adapters{}
	demands := map[adapter.Key]map[string]any{}
	cfgs := adapter.ProcessedConfigsMap{}
	for _, key := range keys {
		adaptersCfg[key] = schema.Adapter{Version: "1.0.0", SDKVersion: "1.0.0"}
		demands[key] = map[string]any{"token": "token"}
		cfgs[key] = map[string]any{}
	}

	auctionStart := time.Now().Add(-30 * time.Second).UnixMilli()
	_, err := builder.HoldAuction(context.Background(), &bidding.BuildParams{
		App:             testApp(4),
		AdapterConfigs:  cfgs,
		BiddingAdapters: keys,
		StartTS:         auctionStart,
		Country:         "DE",
		SessionID:       "sess-dsp",
		AdType:          "banner",
		AdFormat:        "BANNER",
		AuctionRequest: schema.AuctionRequest{
			AdType: "banner",
			AdObject: schema.AdObject{
				AuctionID:  "auc-dsp",
				PriceFloor: 0.01,
				Demands:    demands,
			},
			Adapters: adaptersCfg,
			BaseRequest: schema.BaseRequest{
				Session: schema.Session{ID: "sess-dsp"},
			},
		},
	})
	if err != nil {
		t.Fatalf("HoldAuction() error = %v", err)
	}

	recs := engine.Records()
	var sent, received, rejected int
	outcomes := map[string]string{}
	for _, rec := range recs {
		switch rec.EventName {
		case telemetry.EventDSPRequestSent:
			sent++
		case telemetry.EventDSPResponseReceived:
			received++
			outcomes[rec.DSP] = rec.Outcome
			if rec.LatencyMS < 0 || rec.LatencyMS > 5_000 {
				t.Errorf("%s latency_ms = %d; want send→return, not auction-start→return", rec.DSP, rec.LatencyMS)
			}
			if rec.AuctionID != "auc-dsp" || rec.Country != "DE" {
				t.Errorf("envelope = %+v", rec.Envelope)
			}
		case telemetry.EventDSPResponseRejected:
			rejected++
		}
	}

	if sent != 3 || received != 3 {
		t.Fatalf("DSP events sent=%d received=%d rejected=%d (want 3+3); full auction is 1+3+3+1 with Service.Run", sent, received, rejected)
	}
	if outcomes[string(adapter.BidmachineKey)] != telemetry.OutcomeBid {
		t.Errorf("bidmachine outcome = %q", outcomes[string(adapter.BidmachineKey)])
	}
	if outcomes[string(adapter.MetaKey)] != telemetry.OutcomeNoBid {
		t.Errorf("meta outcome = %q", outcomes[string(adapter.MetaKey)])
	}
	if outcomes[string(adapter.VungleKey)] != telemetry.OutcomeTimeout {
		t.Errorf("vungle outcome = %q", outcomes[string(adapter.VungleKey)])
	}
}

func TestBuilder_HoldAuction_EmitsDSPRejectedBelowFloor(t *testing.T) {
	engine := &telemetry.MemoryEngine{}
	builder := &bidding.Builder{
		AdaptersBuilder: &mocks.AdaptersBuilderMock{
			BuildFunc: func(_ adapter.Key, _ adapter.ProcessedConfigsMap) (*adapters.Bidder, error) {
				return &adapters.Bidder{
					Adapter: scriptedAdapter{response: &adapters.DemandResponse{
						DemandID: adapter.BidmachineKey,
						Status:   http.StatusOK,
						Bid:      &adapters.DemandBid{Price: 0.01},
					}},
					Client: http.DefaultClient,
				}, nil
			},
		},
		NotificationHandler: &mocks.NotificationHandlerMock{
			HandleBiddingRoundFunc: func(context.Context, *schema.AdObject, bidding.AuctionResult, string, string) error {
				return nil
			},
		},
		BidCacher: &mocks.BidCacherMock{
			ApplyBidCacheFunc: func(_ context.Context, _ *schema.AuctionRequest, aucRes *bidding.AuctionResult) []adapters.DemandResponse {
				return aucRes.Bids
			},
		},
		Telemetry: &telemetry.Logger{Engine: engine},
	}

	_, err := builder.HoldAuction(context.Background(), &bidding.BuildParams{
		App:             testApp(4),
		AdapterConfigs:  adapter.ProcessedConfigsMap{adapter.BidmachineKey: map[string]any{}},
		BiddingAdapters: []adapter.Key{adapter.BidmachineKey},
		AuctionRequest: schema.AuctionRequest{
			AdType: "banner",
			AdObject: schema.AdObject{
				AuctionID:  "auc-below-floor",
				PriceFloor: 1.0,
				Demands:    map[adapter.Key]map[string]any{adapter.BidmachineKey: {"token": "token"}},
			},
			Adapters: schema.Adapters{adapter.BidmachineKey: {Version: "1.0.0", SDKVersion: "1.0.0"}},
		},
	})
	if err != nil {
		t.Fatalf("HoldAuction() error = %v", err)
	}

	var rejected *telemetry.Record
	for _, rec := range engine.Records() {
		if rec.EventName == telemetry.EventDSPResponseRejected {
			r := rec
			rejected = &r
			break
		}
	}
	if rejected == nil {
		t.Fatal("expected dsp_response_rejected for a bid below floor")
	}
	if rejected.RejectReason != telemetry.RejectReasonBelowFloor {
		t.Errorf("reject_reason = %q, want %q", rejected.RejectReason, telemetry.RejectReasonBelowFloor)
	}
	if rejected.Price != 0.01 || rejected.PriceFloor <= rejected.Price {
		t.Errorf("price = %v floor = %v", rejected.Price, rejected.PriceFloor)
	}
}
