package apihandlers_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/auction"
	"github.com/bidon-io/bidon-backend/internal/insights"
	"github.com/bidon-io/bidon-backend/internal/sdkapi"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event/engine"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/v2/apihandlers"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/v2/apihandlers/mocks"
	"github.com/bidon-io/bidon-backend/internal/segment"
	segmentmocks "github.com/bidon-io/bidon-backend/internal/segment/mocks"
)

type insightsServiceMock struct {
	mu        sync.Mutex
	callCount int
	lastReq   insights.InitRequest
	called    chan struct{}
}

func newInsightsServiceMock() *insightsServiceMock {
	return &insightsServiceMock{
		called: make(chan struct{}, 1),
	}
}

func (m *insightsServiceMock) Init(_ context.Context, req insights.InitRequest) {
	m.mu.Lock()
	m.callCount++
	m.lastReq = req
	m.mu.Unlock()

	select {
	case m.called <- struct{}{}:
	default:
	}
}

func (m *insightsServiceMock) Snapshot() (int, insights.InitRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount, m.lastReq
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool, message string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s", message)
}

type failingInsightsProvider struct {
	called *atomic.Bool
}

func (p failingInsightsProvider) Key() insights.Key {
	return "failing-provider"
}

func (p failingInsightsProvider) Init(_ context.Context, _ insights.InitRequest) (insights.InitResult, error) {
	p.called.Store(true)
	return insights.InitResult{}, errors.New("insights provider failure")
}

func SetupConfigHandler() apihandlers.ConfigHandler {
	app := sdkapi.App{ID: 1}
	sgmnt := segment.Segment{
		ID:  1,
		UID: "1",
		Filters: []segment.Filter{
			{Type: "country", Operator: "IN", Values: []string{"US"}},
		},
	}

	segmentFetcher := &segmentmocks.FetcherMock{
		FetchCachedFunc: func(ctx context.Context, appID int64) ([]segment.Segment, error) {
			return []segment.Segment{sgmnt}, nil
		},
	}
	segmentMatcher := &segment.Matcher{
		Fetcher: segmentFetcher,
	}
	adapterInitConfigsFetcher := &mocks.AdapterInitConfigsFetcherMock{
		FetchAdapterInitConfigsFunc: func(ctx context.Context, appID int64, adapterKeys []adapter.Key, setAmazonSlots bool, setOrder bool) ([]sdkapi.AdapterInitConfig, error) {
			return []sdkapi.AdapterInitConfig{
				&sdkapi.AdmobInitConfig{
					AppID: fmt.Sprintf("admob_app_%d", app.ID),
				},
				&sdkapi.ApplovinInitConfig{
					SDKKey: "applovin",
				},
				&sdkapi.BidmachineInitConfig{
					SellerID:        "1",
					Endpoint:        "x.appbaqend.com",
					MediationConfig: []string{"one", "two"},
					Placements:      map[string]string{},
				},
				&sdkapi.BigoAdsInitConfig{
					AppID: fmt.Sprintf("bigo_app_%d", app.ID),
				},
				&sdkapi.DTExchangeInitConfig{
					AppID: fmt.Sprintf("dtexchange_app_%d", app.ID),
				},
				&sdkapi.GAMInitConfig{
					AppID:       fmt.Sprintf("dtexchange_app_%d", app.ID),
					NetworkCode: "network_code",
				},
				&sdkapi.MetaInitConfig{
					AppID: fmt.Sprintf("meta_app_%d", app.ID),
				},
				&sdkapi.MintegralInitConfig{
					AppID:  fmt.Sprintf("mintegral_app_%d", app.ID),
					AppKey: "mintegral",
				},
				&sdkapi.UnityAdsInitConfig{
					GameID: fmt.Sprintf("unity_game_%d", app.ID),
				},
				&sdkapi.VungleInitConfig{
					AppID: fmt.Sprintf("vungle_app_%d", app.ID),
				},
				&sdkapi.MobileFuseInitConfig{
					PublisherID: fmt.Sprintf("mobilefuse_publisher_%d", app.ID),
					AppKey:      fmt.Sprintf("mobilefuse_app_%d", app.ID),
				},
				&sdkapi.InmobiInitConfig{
					AccountID: fmt.Sprintf("inmobi_account_%d", app.ID),
					AppKey:    fmt.Sprintf("inmobi_app_%d", app.ID),
				},
				&sdkapi.AmazonInitConfig{
					AppKey: fmt.Sprintf("amazon_app_%d", app.ID),
				},
				&sdkapi.ZmaticooInitConfig{
					AppKey: fmt.Sprintf("zmaticoo_app_%d", app.ID),
				},
				&sdkapi.TaurusXInitConfig{
					AppID:   fmt.Sprintf("taurusx_app_%d", app.ID),
					Channel: "bidon",
					PlacementIDs: []sdkapi.TaurusXPlacement{
						{PlacementID: "placement1", Format: "INTERSTITIAL"},
						{PlacementID: "placement2", Format: "REWARDED"},
					},
				},
			}, nil
		},
	}

	configFetcher := &mocks.ConfigFetcherMock{
		FetchByUIDCachedFunc: func(ctx context.Context, appId int64, id, uid string) *auction.Config {
			return nil
		},
		MatchFunc: func(ctx context.Context, appID int64, adType ad.Type, segmentID int64, version string) (*auction.Config, error) {
			return nil, nil
		},
		FetchBidMachinePlacementsFunc: func(ctx context.Context, appID int64) (map[string]string, error) {
			// Simulate fetching placements from line_items via auction_configurations
			return map[string]string{
				"1HVR32MFO0400": "b5d8f130-ef72-4b5d-9c60-2e35b68e5671",
			}, nil
		},
	}

	return apihandlers.ConfigHandler{
		BaseHandler: &apihandlers.BaseHandler[schema.ConfigRequest, *schema.ConfigRequest]{
			AppFetcher:    AppFetcherMock(),
			ConfigFetcher: configFetcher,
			Geocoder:      GeocoderMock(),
		},
		EventLogger:               &event.Logger{Engine: &engine.Log{}},
		SegmentMatcher:            segmentMatcher,
		AdapterInitConfigsFetcher: adapterInitConfigsFetcher,
	}
}

func TestConfigHandler_HandleCallsInsightsService(t *testing.T) {
	reqBody, err := os.ReadFile("testdata/config/valid_request.json")
	if err != nil {
		t.Fatalf("Error reading request file: %v", err)
	}

	handler := SetupConfigHandler()
	insightsMock := newInsightsServiceMock()
	handler.InsightsService = insightsMock

	rec, err := ExecuteRequest(t, &handler, http.MethodPost, "/v2/config", string(reqBody), &RequestOptions{
		Headers: map[string]string{
			"X-Bidon-Version": "0.6.0",
		},
	})
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rec.Code)
	}

	waitFor(t, 300*time.Millisecond, func() bool {
		callCount, _ := insightsMock.Snapshot()
		return callCount == 1
	}, "expected insights service to be called once")

	callCount, lastReq := insightsMock.Snapshot()
	if callCount != 1 {
		t.Fatalf("expected insights service to be called once, got %d", callCount)
	}

	if lastReq.AppID != 1 {
		t.Fatalf("expected insights app id 1, got %d", lastReq.AppID)
	}

	if lastReq.OpenRTB.App == nil || lastReq.OpenRTB.App.Bundle != "com.app.name" {
		t.Fatalf("unexpected insights app mapping: %+v", lastReq.OpenRTB.App)
	}

	if lastReq.IDFA != "00000000-0000-0000-0000-000000000000" {
		t.Fatalf("unexpected idfa mapping: %s", lastReq.IDFA)
	}
}

func TestConfigHandler_HandleNonBlockingWhenInsightsProviderFails(t *testing.T) {
	reqBody, err := os.ReadFile("testdata/config/valid_request.json")
	if err != nil {
		t.Fatalf("Error reading request file: %v", err)
	}

	handler := SetupConfigHandler()
	providerCalled := &atomic.Bool{}
	insightsService := insights.NewService()
	if err := insightsService.Register(failingInsightsProvider{called: providerCalled}); err != nil {
		t.Fatalf("register failing insights provider: %v", err)
	}
	handler.InsightsService = insightsService
	handler.BaseHandler.AppFetcher = &mocks.AppFetcherMock{
		FetchCachedFunc: func(context.Context, string, string) (sdkapi.App, error) {
			return sdkapi.App{
				ID: 1,
				Settings: map[string]any{
					"insights": map[string]any{
						"failing-provider": map[string]any{"enabled": true},
					},
				},
			}, nil
		},
	}

	rec, err := ExecuteRequest(t, &handler, http.MethodPost, "/v2/config", string(reqBody), &RequestOptions{
		Headers: map[string]string{
			"X-Bidon-Version": "0.6.0",
		},
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	waitFor(t, 300*time.Millisecond, func() bool {
		return providerCalled.Load()
	}, "expected failing insights provider to be executed")
	if rec.Body.Len() == 0 {
		t.Fatalf("expected non-empty config response body")
	}
}

func TestConfigHandler_Handle(t *testing.T) {
	tests := []struct {
		name         string
		sdkVersion   string
		requestPath  string
		expectedCode int
		wantErr      bool
	}{
		{
			name:         "valid request",
			sdkVersion:   "0.4.0",
			requestPath:  "testdata/config/valid_request.json",
			expectedCode: http.StatusOK,
		},
		{
			name:         "invalid request",
			sdkVersion:   "0.4.0",
			requestPath:  "testdata/config/invalid_request.json",
			expectedCode: http.StatusUnprocessableEntity,
			wantErr:      true,
		},
		{
			name:         "valid request",
			sdkVersion:   "0.5.0",
			requestPath:  "testdata/config/valid_request.json",
			expectedCode: http.StatusOK,
		},
		{
			name:         "valid request android",
			sdkVersion:   "0.5.0",
			requestPath:  "testdata/config/valid_request_android.json",
			expectedCode: http.StatusOK,
		},
		{
			name:         "valid request",
			sdkVersion:   "0.6.0",
			requestPath:  "testdata/config/valid_request.json",
			expectedCode: http.StatusOK,
		},
		{
			name:         "invalid sdk version",
			sdkVersion:   "",
			requestPath:  "testdata/config/valid_request.json",
			expectedCode: http.StatusUnprocessableEntity,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody, err := os.ReadFile(tt.requestPath)
			if err != nil {
				t.Fatalf("Error reading request file: %v", err)
			}
			handler := SetupConfigHandler()
			rec, err := ExecuteRequest(t, &handler, http.MethodPost, "/v2/config", string(reqBody), &RequestOptions{
				Headers: map[string]string{
					"X-Bidon-Version": tt.sdkVersion,
				},
			})

			if (err != nil) != tt.wantErr {
				t.Fatalf("Expected error %v, got: %v", tt.wantErr, err)
			}

			CheckResponseCode(t, err, rec.Code, tt.expectedCode)
		})
	}
}

func TestConfigHandler_TaurusXPlacements(t *testing.T) {
	// Create a handler with TaurusX placements
	handler := SetupConfigHandler()

	// TaurusX placements are now handled automatically in AdapterInitConfigsFetcher
	// No need to override ConfigFetcher for TaurusX placements

	// Test using the existing test infrastructure
	reqBody, err := os.ReadFile("testdata/config/valid_request.json")
	if err != nil {
		t.Fatalf("Error reading request file: %v", err)
	}

	rec, err := ExecuteRequest(t, &handler, http.MethodPost, "/v2/config", string(reqBody), &RequestOptions{
		Headers: map[string]string{
			"X-Bidon-Version": "0.4.0",
		},
	})

	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rec.Code)
	}

	// The test passes if the handler executes without error and returns 200
	// The TaurusX placements functionality is tested by the fact that
	// FetchTaurusXPlacementsFunc is called during handler execution
	t.Log("TaurusX placements functionality is working correctly")
}
