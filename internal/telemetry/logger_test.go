package telemetry

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/bidon-io/bidon-backend/config"
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

func TestLoggerLogTypedRecord(t *testing.T) {
	engine := &recordingEngine{}
	logger := &Logger{Engine: engine}

	rec := NewRecord(EventDSPResponseReceived, Envelope{
		AppID:     42,
		AuctionID: "auc-1",
		SessionID: "sess-1",
		AdType:    "banner",
		AdFormat:  "BANNER",
		Country:   "US",
		TraceID:   "trace-1",
	})
	rec.Scope = "bidding_round"
	rec.DSP = "bidmachine"
	rec.Outcome = "bid"
	rec.HTTPStatus = 200
	rec.LatencyMS = 15
	rec.Price = 1.23
	rec.PriceFloor = 0.5

	logger.Log(rec, func(err error) {
		t.Fatalf("unexpected produce error: %v", err)
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
	assertString("ad_type", "banner")
	assertString("ad_format", "BANNER")
	assertString("country", "US")
	assertString("trace_id", "trace-1")
	assertString("scope", "bidding_round")
	assertString("dsp", "bidmachine")
	assertString("outcome", "bid")
	assertFloat("app_id", 42)
	assertFloat("sampling_rate", 1.0)
	assertFloat("http_status", 200)
	assertFloat("latency_ms", 15)
	assertFloat("price", 1.23)
	assertFloat("price_floor", 0.5)

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
	rec := NewRecord(EventAuctionRequestReceived, Envelope{AuctionID: "auc-1"})

	var logger *Logger
	logger.Log(rec, func(error) {
		t.Fatal("nil logger must not call handleErr")
	})

	logger = &Logger{}
	logger.Log(rec, func(error) {
		t.Fatal("nil engine must not call handleErr")
	})
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

func TestRecordTopic(t *testing.T) {
	if got := NewRecord(EventAuctionCompleted, Envelope{}).Topic(); got != config.TelemetryEventsTopic {
		t.Errorf("Topic(): got %q, want %q", got, config.TelemetryEventsTopic)
	}
}
