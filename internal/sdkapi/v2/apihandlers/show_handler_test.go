package apihandlers_test

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/bidon-io/bidon-backend/internal/db"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event/engine"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/v2/apihandlers"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/v2/apihandlers/mocks"
	"github.com/bidon-io/bidon-backend/internal/telemetry"
	telemetryv1 "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/telemetry/v1"
)

func SetupShowHandler() apihandlers.ShowHandler {
	mockHandler := &mocks.ShowNotificationHandlerMock{}
	mockHandler.HandleShowFunc = func(ctx context.Context, imp *schema.Bid, _ string, _ string) {}

	mockAdUnitLookup := &mocks.AdUnitLookupMock{}
	mockAdUnitLookup.GetByUIDCachedFunc = func(ctx context.Context, uid string) (*db.LineItem, error) {
		// Return a test LineItem based on the UID
		if uid == "test_uid_123" {
			return &db.LineItem{
				ID: 456,
				Extra: map[string]any{
					"api_key":   "sk_test_123",
					"placement": "banner_main",
					"timeout":   30,
					"enabled":   true,
				},
			}, nil
		}
		return nil, nil
	}

	return apihandlers.ShowHandler{
		BaseHandler: &apihandlers.BaseHandler[schema.ShowRequest, *schema.ShowRequest]{
			AppFetcher: AppFetcherMock(),
			Geocoder:   GeocoderMock(),
		},
		EventLogger:         &event.Logger{Engine: &engine.Log{}},
		NotificationHandler: mockHandler,
		AdUnitLookup:        mockAdUnitLookup,
	}
}

func TestShowHandler_Handle(t *testing.T) {
	tests := []struct {
		name         string
		requestPath  string
		expectedCode int
		wantErr      bool
	}{
		{
			name:         "valid request",
			requestPath:  "testdata/show/valid_request.json",
			expectedCode: http.StatusOK,
		},
		{
			name:         "invalid request",
			requestPath:  "testdata/show/invalid_request.json",
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
			handler := SetupShowHandler()
			rec, err := ExecuteRequest(t, &handler, http.MethodPost, "/v2/show", string(reqBody), nil)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Expected error %v, got: %v", tt.wantErr, err)
			}

			CheckResponseCode(t, err, rec.Code, tt.expectedCode)
		})
	}
}

func TestShowHandler_EmitsAdImpression(t *testing.T) {
	reqBody, err := os.ReadFile("testdata/show/valid_request.json")
	if err != nil {
		t.Fatalf("Error reading request file: %v", err)
	}
	engine := &telemetry.MemoryEngine{}
	handler := SetupShowHandler()
	handler.Telemetry = telemetry.New(engine, nil)
	rec, err := ExecuteRequest(t, &handler, http.MethodPost, "/v2/show", string(reqBody), nil)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	CheckResponseCode(t, err, rec.Code, http.StatusOK)

	mock := handler.NotificationHandler.(*mocks.ShowNotificationHandlerMock)
	if len(mock.HandleShowCalls()) != 1 {
		t.Fatalf("HandleShow calls = %d, want 1", len(mock.HandleShowCalls()))
	}
	imps := telemetry.EventsOf[*telemetryv1.AdImpression](engine)
	if len(imps) != 1 {
		t.Fatalf("ad_impression = %d, want 1", len(imps))
	}
	if imps[0].GetDsp() != "bigoads" {
		t.Errorf("dsp = %q, want bigoads", imps[0].GetDsp())
	}
	if imps[0].GetEnvelope().GetAuctionId() != "93be3de1-3cf7-4537-9b5e-8c8cf16c2391" {
		t.Errorf("auction_id = %q", imps[0].GetEnvelope().GetAuctionId())
	}
	if imps[0].GetEnvelope().GetAppId() != 1 {
		t.Errorf("app_id = %d, want 1", imps[0].GetEnvelope().GetAppId())
	}
	env := imps[0].GetEnvelope()
	if env.GetAdvertisingIdState() != telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_ZEROED_LAT {
		t.Errorf("advertising_id_state = %s", env.GetAdvertisingIdState())
	}
	if env.GetAdvertisingId() != "00000000-0000-0000-0000-000000000000" {
		t.Errorf("advertising_id = %q", env.GetAdvertisingId())
	}
}
