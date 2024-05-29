package handlers_test

import (
	"context"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/v2/handlers"
	"net/http"
	"os"
	"testing"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/sdkapi"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event/engine"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/v2/handlers/mocks"
	"github.com/bidon-io/bidon-backend/internal/segment"
	segmentmocks "github.com/bidon-io/bidon-backend/internal/segment/mocks"
)

func SetupConfigHandler() handlers.ConfigHandler {
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
		FetchAdapterInitConfigsFunc: func(ctx context.Context, appID int64, adapterKeys []adapter.Key, sdkVersion *semver.Version, setOrder bool) ([]handlers.AdapterInitConfig, error) {
			return []handlers.AdapterInitConfig{
				&handlers.AdmobInitConfig{
					AppID: fmt.Sprintf("admob_app_%d", app.ID),
				},
				&handlers.ApplovinInitConfig{
					AppKey: "applovin",
					SDKKey: "applovin",
				},
				&handlers.BidmachineInitConfig{
					SellerID:        "1",
					Endpoint:        "x.appbaqend.com",
					MediationConfig: []string{"one", "two"},
				},
				&handlers.BigoAdsInitConfig{
					AppID: fmt.Sprintf("bigo_app_%d", app.ID),
				},
				&handlers.DTExchangeInitConfig{
					AppID: fmt.Sprintf("dtexchange_app_%d", app.ID),
				},
				&handlers.GAMInitConfig{
					AppID:       fmt.Sprintf("dtexchange_app_%d", app.ID),
					NetworkCode: "network_code",
				},
				&handlers.MetaInitConfig{
					AppID:     fmt.Sprintf("meta_app_%d", app.ID),
					AppSecret: fmt.Sprintf("meta_app_%d_secret", app.ID),
				},
				&handlers.MintegralInitConfig{
					AppID:  fmt.Sprintf("mintegral_app_%d", app.ID),
					AppKey: "mintegral",
				},
				&handlers.UnityAdsInitConfig{
					GameID: fmt.Sprintf("unity_game_%d", app.ID),
				},
				&handlers.VungleInitConfig{
					AppID: fmt.Sprintf("vungle_app_%d", app.ID),
				},
				&handlers.MobileFuseInitConfig{
					PublisherID: fmt.Sprintf("mobilefuse_publisher_%d", app.ID),
					AppKey:      fmt.Sprintf("mobilefuse_app_%d", app.ID),
				},
				&handlers.InmobiInitConfig{
					AccountID: fmt.Sprintf("inmobi_account_%d", app.ID),
					AppKey:    fmt.Sprintf("inmobi_app_%d", app.ID),
				},
				&handlers.AmazonInitConfig{
					AppKey: fmt.Sprintf("amazon_app_%d", app.ID),
				},
			}, nil
		},
	}

	return handlers.ConfigHandler{
		BaseHandler: &handlers.BaseHandler[schema.ConfigRequest, *schema.ConfigRequest]{
			AppFetcher: AppFetcherMock(),
			Geocoder:   GeocoderMock(),
		},
		EventLogger:               &event.Logger{Engine: &engine.Log{}},
		SegmentMatcher:            segmentMatcher,
		AdapterInitConfigsFetcher: adapterInitConfigsFetcher,
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
			rec, err := ExecuteRequest(t, &handler, http.MethodPost, "/config", string(reqBody), &RequestOptions{
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
