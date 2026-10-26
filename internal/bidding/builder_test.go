package bidding_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/prebid/openrtb/v19/openrtb2"
	dto "github.com/prometheus/client_model/go"
	"go.uber.org/goleak"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

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
	telemetryv1 "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/telemetry/v1"
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
				Telemetry:           telemetry.Nop,
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

func (a scriptedAdapter) ExecuteOptions(openrtb.BidRequest) (adapters.ExecuteRTBOptions, error) {
	return adapters.ExecuteRTBOptions{URL: "http://scripted.invalid/bid"}, nil
}

// ParseBids attaches the scripted bid to the response built from the scripted HTTP reply.
func (a scriptedAdapter) ParseBids(dr *adapters.DemandResponse) (*adapters.DemandResponse, error) {
	dr.Bid = a.response.Bid
	return dr, nil
}

// client returns an HTTP client that replays the scripted response without a network:
// the scripted Error becomes a transport error, otherwise the scripted Status is returned.
func (a scriptedAdapter) client() *http.Client {
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if a.response.Error != nil {
			return nil, a.response.Error
		}
		return &http.Response{
			StatusCode: a.response.Status,
			Body:       io.NopCloser(strings.NewReader("{}")),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	})}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

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
				scripted := scriptedAdapter{response: responses[key]}
				return &adapters.Bidder{Adapter: scripted, Client: scripted.client()}, nil
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
		Telemetry: telemetry.New(engine, nil),
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
	result, err := builder.HoldAuction(context.Background(), &bidding.BuildParams{
		App:             testApp(4),
		AdapterConfigs:  cfgs,
		BiddingAdapters: keys,
		StartTS:         auctionStart,
		Country:         "DE",
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

	for _, bid := range result.Bids {
		if bid.StartTS != auctionStart {
			t.Errorf("%s StartTS = %d, want auction start %d (ad-events timing_map must not change)", bid.DemandID, bid.StartTS, auctionStart)
		}
		if bid.SendTS < auctionStart {
			t.Errorf("%s SendTS = %d, want set at send", bid.DemandID, bid.SendTS)
		}
	}

	sent := telemetry.EventsOf[*telemetryv1.DspRequestSent](engine)
	received := telemetry.EventsOf[*telemetryv1.DspResponseReceived](engine)
	rejected := telemetry.EventsOf[*telemetryv1.DspResponseRejected](engine)
	outcomes := map[string]telemetryv1.Outcome{}
	for _, ev := range received {
		outcomes[ev.GetDsp()] = ev.GetOutcome()
		if ev.GetLatencyMs() < 0 || ev.GetLatencyMs() > 5_000 {
			t.Errorf("%s latency_ms = %d; want send→return, not auction-start→return", ev.GetDsp(), ev.GetLatencyMs())
		}
		if env := ev.GetEnvelope(); env.GetAuctionId() != "auc-dsp" || env.GetCountry() != "DE" {
			t.Errorf("envelope = %v", env)
		}
	}

	if len(sent) != 3 || len(received) != 3 {
		t.Fatalf("DSP events sent=%d received=%d rejected=%d (want 3+3); full auction is 1+3+3+1 with Service.Run", len(sent), len(received), len(rejected))
	}
	if outcomes[string(adapter.BidmachineKey)] != telemetryv1.Outcome_OUTCOME_BID {
		t.Errorf("bidmachine outcome = %v", outcomes[string(adapter.BidmachineKey)])
	}
	if outcomes[string(adapter.MetaKey)] != telemetryv1.Outcome_OUTCOME_NOBID {
		t.Errorf("meta outcome = %v", outcomes[string(adapter.MetaKey)])
	}
	if outcomes[string(adapter.VungleKey)] != telemetryv1.Outcome_OUTCOME_TIMEOUT {
		t.Errorf("vungle outcome = %v", outcomes[string(adapter.VungleKey)])
	}
}

func TestBuilder_HoldAuction_EmitsDSPRejectedBelowFloor(t *testing.T) {
	engine := &telemetry.MemoryEngine{}
	builder := &bidding.Builder{
		AdaptersBuilder: &mocks.AdaptersBuilderMock{
			BuildFunc: func(_ adapter.Key, _ adapter.ProcessedConfigsMap) (*adapters.Bidder, error) {
				scripted := scriptedAdapter{response: &adapters.DemandResponse{
					DemandID: adapter.BidmachineKey,
					Status:   http.StatusOK,
					Bid:      &adapters.DemandBid{Price: 0.01},
				}}
				return &adapters.Bidder{Adapter: scripted, Client: scripted.client()}, nil
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
		Telemetry: telemetry.New(engine, nil),
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

	rejectedEvents := telemetry.EventsOf[*telemetryv1.DspResponseRejected](engine)
	if len(rejectedEvents) == 0 {
		t.Fatal("expected dsp_response_rejected for a bid below floor")
	}
	rejected := rejectedEvents[0]
	if rejected.GetRejectReason() != telemetryv1.RejectReason_REJECT_REASON_BELOW_FLOOR {
		t.Errorf("reject_reason = %v, want below_floor", rejected.GetRejectReason())
	}
	if rejected.GetPrice() != 0.01 || rejected.GetPriceFloor() <= rejected.GetPrice() {
		t.Errorf("price = %v floor = %v", rejected.GetPrice(), rejected.GetPriceFloor())
	}
}

// optionsErrorAdapter fails in ExecuteOptions, the misconfiguration path.
type optionsErrorAdapter struct {
	err error
}

func (a optionsErrorAdapter) BuildImpression(openrtb.BidRequest, *schema.AuctionRequest) (*openrtb2.Imp, adapters.RTBRequestOptions, error) {
	return &openrtb2.Imp{}, adapters.RTBRequestOptions{}, nil
}

func (a optionsErrorAdapter) ExecuteOptions(openrtb.BidRequest) (adapters.ExecuteRTBOptions, error) {
	return adapters.ExecuteRTBOptions{TagID: "tag"}, a.err
}

func TestBuilder_ExecuteOptionsMisconfigurationLogsOnce(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	const cause = "moloco API key is empty"

	adaptersBuilder := &mocks.AdaptersBuilderMock{
		BuildFunc: func(adapter.Key, adapter.ProcessedConfigsMap) (*adapters.Bidder, error) {
			return &adapters.Bidder{
				Adapter: optionsErrorAdapter{err: errors.New(cause)},
				Client:  http.DefaultClient,
			}, nil
		},
	}
	notificationHandler := &mocks.NotificationHandlerMock{
		HandleBiddingRoundFunc: func(context.Context, *schema.AdObject, bidding.AuctionResult, string, string) error {
			return nil
		},
	}
	bidCacher := &mocks.BidCacherMock{
		ApplyBidCacheFunc: func(_ context.Context, _ *schema.AuctionRequest, aucRes *bidding.AuctionResult) []adapters.DemandResponse {
			return aucRes.Bids
		},
	}
	builder := &bidding.Builder{
		AdaptersBuilder:     adaptersBuilder,
		NotificationHandler: notificationHandler,
		BidCacher:           bidCacher,
		Logger:              zap.New(core),
	}

	hold := func(t *testing.T, appID int64, key adapter.Key) adapters.DemandResponse {
		t.Helper()
		result, err := builder.HoldAuction(context.Background(), misconfigParams(appID, key))
		if err != nil {
			t.Fatalf("HoldAuction: %v", err)
		}
		if len(result.Bids) != 1 {
			t.Fatalf("bids: got %d, want 1", len(result.Bids))
		}
		bid := result.Bids[0]
		if !errors.Is(bid.Error, adapters.ErrExecuteOptions) {
			t.Fatalf("error: got %v, want ErrExecuteOptions", bid.Error)
		}
		if bid.Error.Error() != "execute options: "+cause {
			t.Fatalf("error text: got %q", bid.Error.Error())
		}
		if !json.Valid([]byte(bid.RawRequest)) {
			t.Fatalf("RawRequest: got %q, want marshaled bid request", bid.RawRequest)
		}
		return bid
	}

	molocoBefore := optionsErrorCount(t, adapter.MolocoKey)
	bidmachineBefore := optionsErrorCount(t, adapter.BidmachineKey)

	hold(t, 1, adapter.MolocoKey)

	errorLogs := logs.FilterLevelExact(zap.ErrorLevel).FilterMessage("adapter misconfigured").All()
	if len(errorLogs) != 1 {
		t.Fatalf("error logs: got %d, want 1", len(errorLogs))
	}
	assertMisconfigLog(t, errorLogs[0], 1, adapter.MolocoKey, cause)
	if got := optionsErrorCount(t, adapter.MolocoKey); got != molocoBefore+1 {
		t.Fatalf("moloco options errors: got %v, want %v", got, molocoBefore+1)
	}

	hold(t, 1, adapter.MolocoKey)

	errorLogs = logs.FilterLevelExact(zap.ErrorLevel).FilterMessage("adapter misconfigured").All()
	if len(errorLogs) != 1 {
		t.Fatalf("error logs after repeat: got %d, want 1", len(errorLogs))
	}
	debugLogs := logs.FilterLevelExact(zap.DebugLevel).FilterMessage("execute bid request").All()
	if len(debugLogs) != 1 {
		t.Fatalf("debug logs: got %d, want 1", len(debugLogs))
	}
	if got := optionsErrorCount(t, adapter.MolocoKey); got != molocoBefore+2 {
		t.Fatalf("moloco options errors after repeat: got %v, want %v", got, molocoBefore+2)
	}

	hold(t, 1, adapter.BidmachineKey)

	errorLogs = logs.FilterLevelExact(zap.ErrorLevel).FilterMessage("adapter misconfigured").All()
	if len(errorLogs) != 2 {
		t.Fatalf("error logs for a second adapter: got %d, want 2", len(errorLogs))
	}
	assertMisconfigLog(t, errorLogs[1], 1, adapter.BidmachineKey, cause)
	if got := optionsErrorCount(t, adapter.BidmachineKey); got != bidmachineBefore+1 {
		t.Fatalf("bidmachine options errors: got %v, want %v", got, bidmachineBefore+1)
	}

	hold(t, 2, adapter.MolocoKey)

	errorLogs = logs.FilterLevelExact(zap.ErrorLevel).FilterMessage("adapter misconfigured").All()
	if len(errorLogs) != 3 {
		t.Fatalf("error logs for a second app: got %d, want 3", len(errorLogs))
	}
	assertMisconfigLog(t, errorLogs[2], 2, adapter.MolocoKey, cause)
	if got := optionsErrorCount(t, adapter.MolocoKey); got != molocoBefore+3 {
		t.Fatalf("moloco options errors for a second app: got %v, want %v", got, molocoBefore+3)
	}

	hold(t, 2, adapter.MolocoKey)

	errorLogs = logs.FilterLevelExact(zap.ErrorLevel).FilterMessage("adapter misconfigured").All()
	if len(errorLogs) != 3 {
		t.Fatalf("error logs after second app repeat: got %d, want 3", len(errorLogs))
	}
	if got := optionsErrorCount(t, adapter.MolocoKey); got != molocoBefore+4 {
		t.Fatalf("moloco options errors after second app repeat: got %v, want %v", got, molocoBefore+4)
	}
}

func assertMisconfigLog(t *testing.T, entry observer.LoggedEntry, appID int64, key adapter.Key, cause string) {
	t.Helper()
	fields := entry.ContextMap()
	if fields["app_id"] != appID {
		t.Fatalf("app_id: got %#v, want %d", fields["app_id"], appID)
	}
	if fields["demand_id"] != string(key) {
		t.Fatalf("demand_id: got %#v, want %s", fields["demand_id"], key)
	}
	if fields["error"] != "execute options: "+cause {
		t.Fatalf("error field: got %#v", fields["error"])
	}
}

func optionsErrorCount(t *testing.T, key adapter.Key) float64 {
	t.Helper()
	metric, err := bidding.AdapterOptionsErrors.GetMetricWithLabelValues(string(key))
	if err != nil {
		t.Fatalf("adapter_options_errors_total: %v", err)
	}
	pb := &dto.Metric{}
	if err := metric.Write(pb); err != nil {
		t.Fatalf("write metric: %v", err)
	}
	return pb.GetCounter().GetValue()
}

func misconfigParams(appID int64, key adapter.Key) *bidding.BuildParams {
	return &bidding.BuildParams{
		App: testApp(appID),
		AdapterConfigs: adapter.ProcessedConfigsMap{
			key: {"api_key": ""},
		},
		AuctionRequest: schema.AuctionRequest{
			AdObject: schema.AdObject{
				AuctionID: "auc-misconfig",
				Demands: map[adapter.Key]map[string]any{
					key: {"token": "token"},
				},
			},
			Adapters: schema.Adapters{
				key: {Version: "1.0.0", SDKVersion: "1.0.0"},
			},
		},
		BiddingAdapters: []adapter.Key{key},
	}
}
