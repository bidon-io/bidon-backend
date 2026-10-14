package notification_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/prebid/openrtb/v19/openrtb3"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/bidon-io/bidon-backend/internal/notification"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event/engine"
)

func TestHandler_SendEvent(t *testing.T) {
	// Create a test context and input data
	ctx := context.Background()

	// Create a mock HTTP server to handle the LURL request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}

		params := r.URL.Query()
		if diff := cmp.Diff("request-1", params.Get("id")); diff != "" {
			t.Errorf("mismatched id (-want, +got)\n%s", diff)
		}
		if diff := cmp.Diff("4.56", params.Get("auction_price")); diff != "" {
			t.Errorf("mismatched winprice (-want, +got)\n%s", diff)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	bid := notification.Bid{
		RequestID: "request-1",
		ImpID:     "imp-1",
		Price:     1.23,
		LURL:      fmt.Sprintf("%s/lurl?auction_price=${AUCTION_PRICE}&id=${AUCTION_ID}", server.URL),
	}

	// Create a Handler instance with the mock HTTP client and server
	sender := notification.EventSender{
		HttpClient:  server.Client(),
		EventLogger: &event.Logger{Engine: &engine.Log{}},
	}

	p := notification.Params{
		NotificationType: "LURL",
		URL:              bid.LURL,
		Bid:              bid,
		Reason:           openrtb3.LossBelowAuctionFloor,
		FirstPrice:       4.56,
		SecondPrice:      3.00,
	}

	// Call the SendNotificationEvent method with the test context and input data
	sender.SendEvent(ctx, p)
}

type errEngine struct {
	err error
}

func (e errEngine) Produce(_ event.LogMessage, handleErr func(error)) {
	handleErr(e.err)
}

func (e errEngine) Ping(context.Context) error { return nil }

func TestSendEvent_LogsDroppedNotification(t *testing.T) {
	produceErr := errors.New("buffer full")

	tests := []struct {
		name          string
		client        *http.Client
		wantOutcome   string
		wantNotifyErr bool
	}{
		{
			name:        "http ok",
			wantOutcome: "ok",
		},
		{
			name: "http error",
			client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, errors.New("connection refused")
			})},
			wantOutcome:   "error",
			wantNotifyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.client
			template := "https://dsp.example/lurl?id=${AUCTION_ID}"
			if client == nil {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
				t.Cleanup(server.Close)
				client = server.Client()
				template = server.URL + "/lurl?id=${AUCTION_ID}"
			}

			core, logs := observer.New(zap.ErrorLevel)
			sender := notification.EventSender{
				HttpClient:  client,
				EventLogger: &event.Logger{Engine: errEngine{err: produceErr}},
				Logger:      zap.New(core),
			}
			sender.SendEvent(context.Background(), notification.Params{
				Bundle:           "com.example.app",
				AdType:           "interstitial",
				AuctionID:        "auc-1",
				NotificationType: "LURL",
				URL:              template,
				Bid: notification.Bid{
					DemandID:  "applovin",
					ImpID:     "imp-1",
					RequestID: "request-1",
					Price:     1.23,
				},
				Reason: openrtb3.LossBelowAuctionFloor,
			})

			entries := logs.All()
			if len(entries) != 1 {
				t.Fatalf("expected 1 error log, got %d", len(entries))
			}
			if entries[0].Message != "dropped notification event" {
				t.Errorf("message: got %q", entries[0].Message)
			}
			fields := entries[0].ContextMap()
			want := map[string]any{
				"event_type":           "LURL",
				"auction_id":           "auc-1",
				"demand_id":            "applovin",
				"imp_id":               "imp-1",
				"bundle":               "com.example.app",
				"ad_type":              "interstitial",
				"template_url":         template,
				"notification_outcome": tt.wantOutcome,
			}
			for key, value := range want {
				if diff := cmp.Diff(value, fields[key]); diff != "" {
					t.Errorf("%s (-want +got)\n%s", key, diff)
				}
			}
			notifiedURL, _ := fields["url"].(string)
			if notifiedURL == "" || notifiedURL == template {
				t.Errorf("url = %q, want the expanded notice URL", notifiedURL)
			}
			gotProduce, _ := fields["produce_error"].(string)
			if !strings.Contains(gotProduce, produceErr.Error()) {
				t.Errorf("produce_error = %q", gotProduce)
			}
			notify, hasNotify := fields["notification_error"]
			if tt.wantNotifyErr {
				gotNotify, _ := notify.(string)
				if !strings.Contains(gotNotify, "connection refused") {
					t.Errorf("notification_error = %v", notify)
				}
				if gotNotify == gotProduce {
					t.Error("notification_error shadowed by produce_error")
				}
			} else if hasNotify {
				t.Errorf("notification_error = %v, want absent", notify)
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
