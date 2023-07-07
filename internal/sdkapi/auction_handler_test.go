package sdkapi_test

import (
	"context"
	"encoding/json"
	"github.com/bidon-io/bidon-backend/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/auction"
	auctionmocks "github.com/bidon-io/bidon-backend/internal/auction/mocks"
	"github.com/bidon-io/bidon-backend/internal/sdkapi"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/geocoder"
	sdkapimocks "github.com/bidon-io/bidon-backend/internal/sdkapi/mocks"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	"github.com/bidon-io/bidon-backend/internal/segment"
	segmentmocks "github.com/bidon-io/bidon-backend/internal/segment/mocks"
)

func TestAuctionHandler_OK(t *testing.T) {
	app := sdkapi.App{ID: 1}
	geodata := geocoder.GeoData{CountryCode: "US"}
	auctionConfig := &auction.Config{
		ID: 1,
		Rounds: []auction.RoundConfig{
			{
				ID:      "ROUND_1",
				Demands: []adapter.Key{adapter.ApplovinKey, adapter.BidmachineKey},
				Timeout: 15000,
			},
			{
				ID:      "ROUND_2",
				Demands: []adapter.Key{adapter.UnityAdsKey},
				Bidding: []adapter.Key{adapter.BidmachineKey},
				Timeout: 15000,
			},
			{
				ID:      "ROUND_3",
				Demands: []adapter.Key{adapter.ApplovinKey},
				Timeout: 15000,
			},
			{
				ID:      "ROUND_4",
				Demands: []adapter.Key{adapter.UnityAdsKey, adapter.ApplovinKey},
				Timeout: 15000,
			},
			{
				ID:      "ROUND_5",
				Bidding: []adapter.Key{adapter.BidmachineKey},
				Timeout: 15000,
			},
		},
	}
	lineItems := []auction.LineItem{
		{ID: "test", PriceFloor: 0.1, AdUnitID: "test_id"},
	}
	segments := []segment.Segment{
		{ID: 1, Filters: []segment.Filter{{Type: "country", Name: "country", Operator: "IN", Values: []string{"US", "UK"}}}},
	}

	appFetcher := &sdkapimocks.AppFetcherMock{
		FetchFunc: func(ctx context.Context, appKey string, appBundle string) (sdkapi.App, error) {
			return app, nil
		},
	}
	gcoder := &sdkapimocks.GeocoderMock{
		LookupFunc: func(ctx context.Context, ipString string) (geocoder.GeoData, error) {
			return geodata, nil
		},
	}
	configMatcher := &auctionmocks.ConfigMatcherMock{
		MatchFunc: func(ctx context.Context, appID int64, adType ad.Type, segmentID int64) (*auction.Config, error) {
			return auctionConfig, nil
		},
	}
	lineItemsMatcher := &auctionmocks.LineItemsMatcherMock{
		MatchFunc: func(ctx context.Context, params *auction.BuildParams) ([]auction.LineItem, error) {
			return lineItems, nil
		},
	}
	segmentFetcher := &segmentmocks.FetcherMock{
		FetchFunc: func(ctx context.Context, appID int64) ([]segment.Segment, error) {
			return segments, nil
		},
	}
	auctionBuilder := &auction.Builder{
		ConfigMatcher:    configMatcher,
		LineItemsMatcher: lineItemsMatcher,
	}
	segmentMatcher := &segment.Matcher{
		Fetcher: segmentFetcher,
	}

	// Create a new AuctionHandler instance

	handler := &sdkapi.AuctionHandler{
		BaseHandler: &sdkapi.BaseHandler[schema.AuctionRequest, *schema.AuctionRequest]{
			AppFetcher: appFetcher,
			Geocoder:   gcoder,
		},
		AuctionBuilder: auctionBuilder,
		SegmentMatcher: segmentMatcher,
	}

	// Create a sample request body

	requestJson := `{
		"adapters": {
			"admob": {
				"version": "0.2.1.11",
				"sdk_version": "21.5.0"
			},
			"bidmachine": {
				"version": "0.2.1.12",
				"sdk_version": "2.1.13"
			},
			"dtexchange": {
				"version": "0.2.1.11",
				"sdk_version": "8.2.3"
			},
			"unityads": {
				"version": "0.2.1.11",
				"sdk_version": "4.5.0"
			}
		},
		"device": {
			"connection_type": "WIFI",
			"model": "Google Pixel 3",
			"hwv": "blueline",
			"h": 2028,
			"js": 1,
			"language": "en_US",
			"make": "Google",
			"os": "android",
			"os_api_level": "31",
			"osv": "12",
			"ppi": 440,
			"pxratio": 2.75,
			"type": "PHONE",
			"ua": "Mozilla\/5.0 (Linux; Android 12; Pixel 3 Build\/SP1A.210812.016.C1; wv) AppleWebKit\/537.36 (KHTML, like Gecko) Version\/4.0 Chrome\/114.0.5735.131 Mobile Safari\/537.36",
			"w": 1080
		},
		"app": {
			"bundle": "com.newpubco.merge",
			"framework": "unity",
			"framework_version": "2020.3.48f1",
			"key": "23f1181b191f34a04f4a74840c09d116028a98ed3721de6b",
			"version": "2.6.42"
		},
		"token": "{}",
		"session": {
			"battery": 100,
			"cpu_usage": 0.7714286,
			"id": "d0b9ec0e-60e9-4501-bcbc-67712562dab9",
			"launch_monotonic_ts": 68406545,
			"launch_ts": 1687863971253,
			"memory_warnings_monotonic_ts": [],
			"memory_warnings_ts": [],
			"start_monotonic_ts": 68406545,
			"monotonic_ts": 68471084,
			"ram_size": 3753299968,
			"ram_used": 413565952,
			"start_ts": 1687863971253,
			"storage_free": 30742986752,
			"storage_used": 24636485632,
			"ts": 1687864035792
		},
		"user": {
			"idg": "8020a0a6-5e55-4b8f-af9a-7b01a9aad4fc",
			"coppa": false,
			"idfa": "0d95325a-71b7-4334-8816-9e2935ec0eef",
			"tracking_authorization_status": "AUTHORIZED"
		},
		"ext": "{\"appodeal_segment_id\":20194,\"appodeal_session_id\":\"c6c5ce3e-b8cd-492c-ae7f-15f0e679883b\",\"appodeal_token\":{\"id\":1687857767,\"last_init\":1687863971,\"signature\":\"RXVQdjhUdUE4Q1UyZkVKVkdYN3hKMGU0YzREa29JNGFleDk3SFg0NmlmS2FuWDF0bFVVSUpZWVVJZFE3ZUIvZy0tRHlmSFNNT2Q3M01MQU9zbWc5L2o4QT09--a9b3ccac217e3a2f4be477f2f15dbb5785c0747f\"},\"ext\":{\"sample_key\":\"sample_value\"}}",
		"ad_object": {
			"auction_id": "fe91db30-013d-40ff-b9b8-2e99519dd5e0",
			"interstitial": {},
			"orientation": "PORTRAIT",
			"pricefloor": 0.66
		}
	}`

	// Create a new HTTP request
	req := httptest.NewRequest(http.MethodPost, "/auction/interstitial", strings.NewReader(requestJson))
	req.Header.Set("Content-Type", "application/json")

	// Create a new HTTP response recorder
	rec := httptest.NewRecorder()

	// Create a new Echo instance and context
	e := config.Echo("sdkapi-test", nil)
	c := e.NewContext(req, rec)

	// Call the Handle method
	err := handler.Handle(c)
	if err != nil {
		t.Fatalf("Handle method returned an error: %v", err)
	}

	// Assert that the response status code is HTTP 200 OK
	assert.Equal(t, http.StatusOK, rec.Code)

	expectedResponseJson := `{
      "auction_configuration_id": 1,
      "external_win_notifications": false,
      "rounds": [
        {
          "id": "ROUND_1",
          "demands": [
            "bidmachine"
          ],
          "bidding": [],
          "timeout": 15000
        },
        {
          "id": "ROUND_2",
          "demands": [
            "unityads"
          ],
          "bidding": [
            "bidmachine"
          ],
          "timeout": 15000
        },
        {
          "id": "ROUND_4",
          "demands": [
            "unityads"
          ],
          "bidding": [],
          "timeout": 15000
        },
        {
          "id": "ROUND_5",
          "demands": [],
          "bidding": [
            "bidmachine"
          ],
          "timeout": 15000
        }
      ],
      "line_items": [
        {
          "id": "test",
          "pricefloor": 0.1,
          "ad_unit_id": "test_id"
        }
      ],
      "segment": {
        "id": "1"
      },
      "token": "{}",
      "pricefloor": 0.66,
      "auction_id": "fe91db30-013d-40ff-b9b8-2e99519dd5e0"
    }`

	var actualResponse interface{}
	var exptectedResponse interface{}

	err = json.Unmarshal([]byte(rec.Body.String()), &actualResponse)
	if err != nil {
		t.Fatalf("Failed to parse JSON1: %s", err)
	}

	err = json.Unmarshal([]byte(expectedResponseJson), &exptectedResponse)
	if err != nil {
		t.Fatalf("Failed to parse JSON2: %s", err)
	}

	assert.Equal(t, actualResponse, exptectedResponse)
}

// TODO: add tests for:
// test ErrNoAdsFound
// test resolve request error
// test Validate error
// test AppFetcher error
