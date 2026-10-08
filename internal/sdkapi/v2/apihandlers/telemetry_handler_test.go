package apihandlers_test

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/bidon-io/bidon-backend/internal/sdkapi/v2/apihandlers"
)

func TestTelemetryHandler_Handle(t *testing.T) {
	valid, err := os.ReadFile("testdata/telemetry/valid_request.json")
	if err != nil {
		t.Fatalf("Error reading request file: %v", err)
	}
	missingApp, err := os.ReadFile("testdata/telemetry/missing_app.json")
	if err != nil {
		t.Fatalf("Error reading request file: %v", err)
	}

	tests := []struct {
		name         string
		body         string
		expectedCode int
		wantErr      bool
		accepted     int
	}{
		{
			name:         "valid request",
			body:         string(valid),
			expectedCode: http.StatusAccepted,
			accepted:     3,
		},
		{
			name:         "unknown event name is dropped",
			body:         `{"app":{"key":"app_abc123","bundle":"com.example.game"},"session":{"id":"550e8400-e29b-41d4-a716-446655440000"},"events":[{"event_id":"11111111-1111-4111-8111-111111111111","event_name":"not_in_catalogue","event_ts":1710000000000,"sampling_rate":1}]}`,
			expectedCode: http.StatusAccepted,
			accepted:     0,
		},
		{
			name:         "missing app",
			body:         string(missingApp),
			expectedCode: http.StatusBadRequest,
			wantErr:      true,
		},
		{
			name:         "invalid json",
			body:         "not-json",
			expectedCode: http.StatusBadRequest,
			wantErr:      true,
		},
		{
			name:         "too many events",
			body:         tooManyEvents(t),
			expectedCode: http.StatusRequestEntityTooLarge,
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zap.InfoLevel)
			handler := apihandlers.TelemetryHandler{Logger: zap.New(core)}
			rec, err := ExecuteRequest(t, &handler, http.MethodPost, "/v2/telemetry", tt.body, &RequestOptions{
				Headers: map[string]string{"X-Bidon-Version": "0.7.0"},
			})

			if (err != nil) != tt.wantErr {
				t.Fatalf("Expected error %v, got: %v", tt.wantErr, err)
			}
			CheckResponseCode(t, err, rec.Code, tt.expectedCode)
			if tt.wantErr {
				return
			}

			CheckResponses(t, []byte(fmt.Sprintf(`{"accepted":%d}`, tt.accepted)), rec.Body.Bytes())
			entries := logs.All()
			if len(entries) != 1 || entries[0].Message != "telemetry ingest" {
				t.Fatalf("logs = %+v", entries)
			}
			if entries[0].ContextMap()["sdk_version"] != "0.7.0" {
				t.Fatalf("sdk_version = %#v", entries[0].ContextMap()["sdk_version"])
			}
		})
	}
}

func tooManyEvents(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	b.WriteString(`{"app":{"key":"app_abc123","bundle":"com.example.game"},"session":{"id":"550e8400-e29b-41d4-a716-446655440000"},"events":[`)
	for i := 0; i < 101; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `{"event_id":"%s","event_name":"auction_requested","event_ts":1}`, eventUUID(i))
	}
	b.WriteString(`]}`)
	return b.String()
}

func eventUUID(i int) string {
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", i+1)
}
