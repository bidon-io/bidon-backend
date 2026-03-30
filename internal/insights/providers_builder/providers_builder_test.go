package providers_builder

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/go-redis/redismock/v9"

	"github.com/bidon-io/bidon-backend/config"
	"github.com/bidon-io/bidon-backend/internal/insights"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/geocoder"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

type loggerEngineMock struct {
	messages []event.LogMessage
}

func (m *loggerEngineMock) Produce(message event.LogMessage, _ func(error)) {
	m.messages = append(m.messages, message)
}

func (m *loggerEngineMock) Ping(context.Context) error {
	return nil
}

func TestBuildReturnsErrorWhenRedisNil(t *testing.T) {
	_, err := Build(Deps{})
	if !errors.Is(err, ErrNilRedisClient) {
		t.Fatalf("expected ErrNilRedisClient, got %v", err)
	}
}

func TestBuildSuccess(t *testing.T) {
	rdb, _ := redismock.NewClusterMock()
	service, err := Build(Deps{Redis: rdb})
	if err != nil {
		t.Fatalf("build insights service: %v", err)
	}
	if service == nil {
		t.Fatalf("expected service to be created")
	}

	// Provider is disabled by default, so this should not perform any Redis calls.
	service.Init(context.Background(), insights.InitRequest{})
}

func TestNewInitResultLoggerEmitsLifecycleEvents(t *testing.T) {
	baseReq := &schema.BaseRequest{
		App: schema.App{
			Bundle:     "com.example.app",
			Version:    "1.0.0",
			SDKVersion: "0.9.0",
		},
		Device: schema.Device{
			OS:    "android",
			Model: "Pixel",
		},
		User: schema.User{
			IDFA: "idfa-value",
			IDG:  "idg-value",
			IDFV: "idfv-value",
		},
	}

	tests := []struct {
		name       string
		result     insights.InitResult
		wantStatus string
	}{
		{
			name: "called success",
			result: insights.InitResult{
				Provider:          insights.NeftaKey,
				RawRequest:        `{"nuid":""}`,
				RawRequestHeaders: `{"nefta-sdk-version":["1.0.0"]}`,
				RawResponse:       `{"nuid":"abc"}`,
				Status:            200,
			},
			wantStatus: "SUCCESS",
		},
		{
			name: "called skipped",
			result: insights.InitResult{
				Provider: insights.NeftaKey,
				Skipped:  true,
			},
			wantStatus: "SKIPPED",
		},
		{
			name: "called failed",
			result: insights.InitResult{
				Provider: insights.NeftaKey,
				Error:    "timeout",
			},
			wantStatus: "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engineMock := &loggerEngineMock{}
			logger := &event.Logger{Engine: engineMock}
			logResult := newInitResultLogger(logger)
			req := insights.InitRequest{
				BaseRequest: baseReq,
				GeoData: geocoder.GeoData{
					CountryCode: "US",
				},
			}

			logResult(req, tt.result)

			if len(engineMock.messages) != 1 {
				t.Fatalf("expected one log message, got %d", len(engineMock.messages))
			}
			msg := engineMock.messages[0]
			if msg.Topic != config.AdEventsTopic {
				t.Fatalf("expected topic %q, got %q", config.AdEventsTopic, msg.Topic)
			}

			var logged event.AdEvent
			if err := json.Unmarshal(msg.Value, &logged); err != nil {
				t.Fatalf("unmarshal logged event: %v", err)
			}

			wantEventType := insightsProviderInitEventType(tt.result.Provider)
			if logged.EventType != wantEventType {
				t.Fatalf("expected event_type %q, got %q", wantEventType, logged.EventType)
			}
			if logged.Status != tt.wantStatus {
				t.Fatalf("expected status %q, got %q", tt.wantStatus, logged.Status)
			}
			if logged.RawRequest != tt.result.RawRequest {
				if tt.result.RawRequestHeaders == "" {
					t.Fatalf("expected raw_request %q, got %q", tt.result.RawRequest, logged.RawRequest)
				}
				if !strings.Contains(logged.RawRequest, `"headers":{"nefta-sdk-version":["1.0.0"]}`) {
					t.Fatalf("expected raw_request to contain headers, got %q", logged.RawRequest)
				}
				if !strings.Contains(logged.RawRequest, `"body":{"nuid":""}`) {
					t.Fatalf("expected raw_request to contain body, got %q", logged.RawRequest)
				}
			}
			if logged.RawResponse != tt.result.RawResponse {
				t.Fatalf("expected raw_response %q, got %q", tt.result.RawResponse, logged.RawResponse)
			}
			if logged.Error != tt.result.Error {
				t.Fatalf("expected error %q, got %q", tt.result.Error, logged.Error)
			}
		})
	}
}

func TestNewFloorPriceResultLoggerEmitsLifecycleEvents(t *testing.T) {
	baseReq := &schema.BaseRequest{
		App: schema.App{
			Bundle:     "com.example.app",
			Version:    "1.0.0",
			SDKVersion: "0.9.0",
		},
		Device: schema.Device{
			OS:    "android",
			Model: "Pixel",
		},
	}

	tests := []struct {
		name       string
		result     insights.FloorPriceResult
		wantStatus string
	}{
		{
			name: "called success",
			result: insights.FloorPriceResult{
				Provider:          insights.NeftaKey,
				Auction:           &insights.FloorPriceRecommendation{AuctionID: 1, FloorPrice: 0.42},
				RawRequest:        `{"nuid":"abc"}`,
				RawRequestHeaders: `{"nefta-sdk-version":["1.0.0"]}`,
				RawResponse:       `{"floor_prices":[]}`,
				Status:            200,
			},
			wantStatus: "SUCCESS",
		},
		{
			name: "called skipped",
			result: insights.FloorPriceResult{
				Provider: insights.NeftaKey,
				Skipped:  true,
			},
			wantStatus: "SKIPPED",
		},
		{
			name: "called failed",
			result: insights.FloorPriceResult{
				Provider: insights.NeftaKey,
				Error:    "timeout",
			},
			wantStatus: "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engineMock := &loggerEngineMock{}
			logger := &event.Logger{Engine: engineMock}
			logResult := newFloorPriceResultLogger(logger)
			req := insights.FloorPriceRequest{
				AuctionID:               "auction-123",
				AuctionConfigurationID:  11,
				AuctionConfigurationUID: 22,
				AdType:                  "rewarded",
				AdFormat:                "VIDEO",
				BaseRequest:             baseReq,
				GeoData: geocoder.GeoData{
					CountryCode: "US",
				},
			}

			logResult(req, tt.result)

			if len(engineMock.messages) != 1 {
				t.Fatalf("expected one log message, got %d", len(engineMock.messages))
			}
			msg := engineMock.messages[0]
			if msg.Topic != config.AdEventsTopic {
				t.Fatalf("expected topic %q, got %q", config.AdEventsTopic, msg.Topic)
			}

			var logged event.AdEvent
			if err := json.Unmarshal(msg.Value, &logged); err != nil {
				t.Fatalf("unmarshal logged event: %v", err)
			}

			wantEventType := insightsProviderFloorPriceEventType(tt.result.Provider)
			if logged.EventType != wantEventType {
				t.Fatalf("expected event_type %q, got %q", wantEventType, logged.EventType)
			}
			if logged.Status != tt.wantStatus {
				t.Fatalf("expected status %q, got %q", tt.wantStatus, logged.Status)
			}
			if tt.name == "called success" {
				if logged.AuctionID != "auction-123" {
					t.Fatalf("expected auction_id auction-123, got %q", logged.AuctionID)
				}
				if logged.AuctionConfigurationID != 11 {
					t.Fatalf("expected auction_configuration_id 11, got %d", logged.AuctionConfigurationID)
				}
				if logged.AuctionConfigurationUID != 22 {
					t.Fatalf("expected auction_configuration_uid 22, got %d", logged.AuctionConfigurationUID)
				}
				if logged.AdFormat != "VIDEO" {
					t.Fatalf("expected ad_format VIDEO, got %q", logged.AdFormat)
				}
				if logged.PriceFloor != 0.42 {
					t.Fatalf("expected price_floor 0.42, got %v", logged.PriceFloor)
				}
			}
		})
	}
}
