package providers_builder

import (
	"context"
	"encoding/json"
	"errors"
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
				Provider:    insights.NeftaKey,
				RawRequest:  `{"nuid":""}`,
				RawResponse: `{"nuid":"abc"}`,
				Status:      200,
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
				t.Fatalf("expected raw_request %q, got %q", tt.result.RawRequest, logged.RawRequest)
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
