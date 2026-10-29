package notification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/prebid/openrtb/v19/openrtb3"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
	"github.com/bidon-io/bidon-backend/internal/telemetry"
	telemetryv1 "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/telemetry/v1"
)

type captureEngine struct {
	mu       sync.Mutex
	messages []event.LogMessage
}

func (e *captureEngine) Produce(message event.LogMessage, _ func(error)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.messages = append(e.messages, message)
}

func (e *captureEngine) Ping(context.Context) error { return nil }

func TestSendEvent_HTTPOutcomes(t *testing.T) {
	tests := []struct {
		name        string
		statuses    []int
		kind        string
		wantStatus  int32
		wantSuccess bool
		wantRetries int32
		wantFailed  bool
	}{
		{name: "500 then 200", statuses: []int{http.StatusInternalServerError, http.StatusOK}, kind: "BURL", wantStatus: 200, wantSuccess: true, wantRetries: 1},
		{name: "500 exhausted", statuses: []int{500, 500, 500, 500}, kind: "BURL", wantStatus: 500, wantSuccess: false, wantRetries: 3, wantFailed: true},
		{name: "400 terminal", statuses: []int{http.StatusBadRequest}, kind: "LURL", wantStatus: 400, wantSuccess: false, wantFailed: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hits int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				status := tt.statuses[len(tt.statuses)-1]
				if hits < len(tt.statuses) {
					status = tt.statuses[hits]
				}
				hits++
				w.WriteHeader(status)
			}))
			t.Cleanup(server.Close)

			tel := &telemetry.MemoryEngine{}
			old := &captureEngine{}
			sender := EventSender{
				HttpClient:  server.Client(),
				EventLogger: &event.Logger{Engine: old},
				Telemetry:   telemetry.New(tel, nil),
				retryBackoff: func() backoff.BackOff {
					return backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Millisecond), 3)
				},
			}
			ctx := telemetry.WithNoticeMeta(context.Background(), telemetry.NoticeMeta{AppID: 9, SessionID: "sess", Country: "US", AdFormat: "banner"})
			sender.SendEvent(ctx, Params{
				Bundle:           "app",
				AdType:           "banner",
				AuctionID:        "auc-notice",
				NotificationType: tt.kind,
				URL:              server.URL + "/notice",
				Bid:              Bid{DemandID: adapter.BidmachineKey, Price: 1.25},
				Reason:           openrtb3.LossBelowAuctionFloor,
				FirstPrice:       1.25,
			})

			if hits != len(tt.statuses) {
				t.Fatalf("http attempts = %d, want %d", hits, len(tt.statuses))
			}
			if len(old.messages) != 1 {
				t.Fatalf("notification-events = %d, want 1", len(old.messages))
			}
			var legacy struct {
				EventType string `json:"event_type"`
				Error     string `json:"error"`
			}
			if err := json.Unmarshal(old.messages[0].Value, &legacy); err != nil {
				t.Fatalf("decode notification event: %v", err)
			}
			if legacy.EventType != tt.kind {
				t.Errorf("event_type = %q, want %q", legacy.EventType, tt.kind)
			}
			if tt.wantSuccess && legacy.Error != "" {
				t.Errorf("legacy error = %q, want empty", legacy.Error)
			}
			if !tt.wantSuccess && legacy.Error == "" {
				t.Error("legacy error empty, want the HTTP failure")
			}

			failed := telemetry.EventsOf[*telemetryv1.NoticeDeliveryFailed](tel)
			if tt.wantFailed && len(failed) != 1 {
				t.Fatalf("notice_delivery_failed = %d, want 1", len(failed))
			}
			if !tt.wantFailed && len(failed) != 0 {
				t.Fatalf("notice_delivery_failed = %d, want 0", len(failed))
			}
			switch tt.kind {
			case "BURL":
				sent := telemetry.EventsOf[*telemetryv1.BillingNoticeSent](tel)
				if len(sent) != 1 {
					t.Fatalf("billing_notice_sent = %d, want 1", len(sent))
				}
				if sent[0].GetSuccess() != tt.wantSuccess || sent[0].GetHttpStatus() != tt.wantStatus || sent[0].GetRetryCount() != tt.wantRetries {
					t.Errorf("billing success=%v status=%d retries=%d", sent[0].GetSuccess(), sent[0].GetHttpStatus(), sent[0].GetRetryCount())
				}
				if sent[0].GetEnvelope().GetAppId() != 9 || sent[0].GetEnvelope().GetAuctionId() != "auc-notice" {
					t.Errorf("envelope = %+v", sent[0].GetEnvelope())
				}
			case "LURL":
				sent := telemetry.EventsOf[*telemetryv1.LossNoticeSent](tel)
				if len(sent) != 1 {
					t.Fatalf("loss_notice_sent = %d, want 1", len(sent))
				}
				if sent[0].GetSuccess() != tt.wantSuccess || sent[0].GetHttpStatus() != tt.wantStatus {
					t.Errorf("loss success=%v status=%d", sent[0].GetSuccess(), sent[0].GetHttpStatus())
				}
			}
		})
	}
}
