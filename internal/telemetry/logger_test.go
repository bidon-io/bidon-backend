package telemetry

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/bidon-io/bidon-backend/config"
	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/sdkapi"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

type recordingEngine struct {
	messages []LogMessage
}

func (e *recordingEngine) Produce(message LogMessage, _ func(error)) {
	e.messages = append(e.messages, message)
}

func (e *recordingEngine) Ping(_ context.Context) error {
	return nil
}

func testAuctionParams() Params {
	return Params{
		Request: &schema.AuctionRequest{
			AdType: ad.BannerType,
			AdObject: schema.AdObject{
				AuctionID: "auc-1",
				Banner:    &schema.BannerAdObject{Format: ad.BannerFormat},
			},
			BaseRequest: schema.BaseRequest{
				Session: schema.Session{ID: "sess-1"},
			},
		},
		App:     &sdkapi.App{ID: 42},
		Country: "US",
		TraceID: "trace-1",
	}
}

func TestEventDSPResponseReceived(t *testing.T) {
	engine := &recordingEngine{}
	logger := New(engine, nil)

	logger.Event.DSPResponseReceived(testAuctionParams(), &adapters.DemandResponse{
		DemandID: adapter.BidmachineKey,
		Status:   http.StatusOK,
		Bid:      &adapters.DemandBid{Price: 1.23},
		StartTS:  0,
		EndTS:    15,
	})

	if len(engine.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(engine.messages))
	}

	msg := engine.messages[0]
	if msg.Topic != config.TelemetryEventsTopic {
		t.Errorf("topic: got %q, want %q", msg.Topic, config.TelemetryEventsTopic)
	}

	var payload map[string]any
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if _, ok := payload["payload"]; ok {
		t.Fatal("typed fields must be named columns, not a payload blob")
	}

	assertString := func(key, want string) {
		t.Helper()
		got, ok := payload[key].(string)
		if !ok || got != want {
			t.Errorf("%s: got %#v, want %q", key, payload[key], want)
		}
	}
	assertFloat := func(key string, want float64) {
		t.Helper()
		got, ok := payload[key].(float64)
		if !ok || got != want {
			t.Errorf("%s: got %#v, want %v", key, payload[key], want)
		}
	}

	for _, key := range []string{"event_id", "event_name", "event_ts", "schema_version", "app_id", "auction_id", "session_id", "sampling_rate"} {
		if _, ok := payload[key]; !ok {
			t.Errorf("missing envelope field %q", key)
		}
	}

	assertString("event_name", EventDSPResponseReceived)
	assertString("schema_version", SchemaVersion)
	assertString("auction_id", "auc-1")
	assertString("session_id", "sess-1")
	assertString("ad_type", string(ad.BannerType))
	assertString("ad_format", string(ad.BannerFormat))
	assertString("country", "US")
	assertString("trace_id", "trace-1")
	assertString("scope", string(ScopeBiddingRound))
	assertString("dsp", string(adapter.BidmachineKey))
	assertString("outcome", string(OutcomeBid))
	assertFloat("app_id", 42)
	assertFloat("sampling_rate", 1.0)
	assertFloat("http_status", 200)
	assertFloat("latency_ms", 15)
	assertFloat("price", 1.23)

	if _, ok := payload["price_floor"]; ok {
		t.Error("dsp_response_received must not include price_floor")
	}

	eventID, _ := payload["event_id"].(string)
	if eventID == "" {
		t.Error("event_id must be a non-empty UUID")
	}
	eventTS, ok := payload["event_ts"].(float64)
	if !ok || eventTS <= 0 {
		t.Errorf("event_ts: got %#v, want unix ms", payload["event_ts"])
	}
}

func TestLoggerNilSafe(t *testing.T) {
	params := testAuctionParams()

	Event{}.AuctionRequestReceived(params)
	New(nil, nil).Event.AuctionRequestReceived(params)
}

func TestKafkaProduceEmptyTopic(t *testing.T) {
	engine := &Kafka{Topics: map[config.Topic]string{}}
	called := false

	engine.Produce(LogMessage{
		Topic: config.TelemetryEventsTopic,
		Value: []byte(`{}`),
	}, func(err error) {
		called = true
		if err == nil {
			t.Fatal("expected error for empty topic")
		}
	})

	if !called {
		t.Fatal("handleErr must be called when topic env is empty")
	}
}

func TestLoggerEmptyTopicStructured(t *testing.T) {
	core, logs := observer.New(zap.ErrorLevel)
	logger := New(&Kafka{Topics: map[config.Topic]string{}}, zap.New(core))

	logger.Event.AuctionRequestReceived(Params{
		Request: &schema.AuctionRequest{
			AdObject: schema.AdObject{AuctionID: "auc-structured"},
			BaseRequest: schema.BaseRequest{
				Session: schema.Session{ID: "sess-structured"},
			},
		},
		App: &sdkapi.App{ID: 7},
	})

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 error log, got %d", len(entries))
	}
	if entries[0].Message != "produce telemetry event" {
		t.Errorf("message: got %q", entries[0].Message)
	}

	fields := entries[0].ContextMap()
	assertField(t, fields, "topic", string(config.TelemetryEventsTopic))
	assertField(t, fields, "event_name", EventAuctionRequestReceived)
	assertField(t, fields, "auction_id", "auc-structured")
	assertField(t, fields, "session_id", "sess-structured")
	if fields["app_id"] != int64(7) {
		t.Errorf("app_id: got %#v, want 7", fields["app_id"])
	}
	if _, ok := fields["error"]; !ok {
		t.Error("expected error field")
	}
}

func TestLogEngineStructured(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	engine := &Log{Logger: zap.New(core)}
	logger := New(engine, nil)

	logger.Event.DSPResponseReceived(Params{
		Request: &schema.AuctionRequest{
			AdObject: schema.AdObject{AuctionID: "auc-log"},
			BaseRequest: schema.BaseRequest{
				Session: schema.Session{ID: "sess-log"},
			},
		},
		App:     &sdkapi.App{ID: 3},
		TraceID: "trace-log",
	}, &adapters.DemandResponse{DemandID: adapter.BidmachineKey})

	entries := logs.FilterMessage("produce telemetry").All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 debug log, got %d", len(entries))
	}

	fields := entries[0].ContextMap()
	assertField(t, fields, "event_name", EventDSPResponseReceived)
	assertField(t, fields, "auction_id", "auc-log")
	assertField(t, fields, "dsp", string(adapter.BidmachineKey))
	assertField(t, fields, "trace_id", "trace-log")
	if _, ok := fields["value"]; ok {
		t.Error("raw JSON blob must not be logged; use named fields")
	}
}

func TestEventNames(t *testing.T) {
	tests := []struct {
		name string
		got  string
	}{
		{EventAuctionRequestReceived, auctionRequestReceived{}.EventName()},
		{EventAuctionCompleted, auctionCompleted{}.EventName()},
		{EventDSPRequestSent, dspRequestSent{}.EventName()},
		{EventDSPResponseReceived, dspResponseReceived{}.EventName()},
		{EventDSPResponseRejected, dspResponseRejected{}.EventName()},
	}
	for _, tt := range tests {
		if tt.got != tt.name {
			t.Errorf("EventName() = %q, want %q", tt.got, tt.name)
		}
	}
}

func assertField(t *testing.T, fields map[string]any, key, want string) {
	t.Helper()
	got, ok := fields[key]
	if !ok || got != want {
		t.Errorf("%s: got %#v, want %q", key, fields[key], want)
	}
}
