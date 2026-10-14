package bidding_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

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
