package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

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

func TestLoggerEmptyTopicStructured(t *testing.T) {
	core, logs := observer.New(zap.ErrorLevel)
	logger := &Logger{
		Engine: &Kafka{Topics: map[config.Topic]string{}},
		Logger: zap.New(core),
	}

	rec := NewRecord(EventAuctionRequestReceived, Envelope{
		AppID:     7,
		AuctionID: "auc-structured",
		SessionID: "sess-structured",
	})
	logger.Log(rec, func(error) {})

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
	logger := &Logger{Engine: engine}

	rec := NewRecord(EventDSPResponseReceived, Envelope{
		AppID:     3,
		AuctionID: "auc-log",
		SessionID: "sess-log",
		TraceID:   "trace-log",
	})
	rec.DSP = "bidmachine"
	logger.Log(rec, func(error) {})

	entries := logs.FilterMessage("produce telemetry").All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 debug log, got %d", len(entries))
	}

	fields := entries[0].ContextMap()
	assertField(t, fields, "event_name", EventDSPResponseReceived)
	assertField(t, fields, "auction_id", "auc-log")
	assertField(t, fields, "dsp", "bidmachine")
	assertField(t, fields, "trace_id", "trace-log")
	if _, ok := fields["value"]; ok {
		t.Error("raw JSON blob must not be logged; use named fields")
	}
}

type brokenEvent struct{}

func (brokenEvent) Topic() config.Topic { return config.TelemetryEventsTopic }

func (brokenEvent) MarshalJSON() ([]byte, error) { return nil, errors.New("boom") }

func TestLoggerMarshalErrorStructured(t *testing.T) {
	core, logs := observer.New(zap.ErrorLevel)
	logger := &Logger{
		Engine: &recordingEngine{},
		Logger: zap.New(core),
	}

	logger.Log(brokenEvent{}, func(error) {})

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 error log, got %d", len(entries))
	}
	if entries[0].Message != "marshal telemetry event" {
		t.Errorf("message: got %q", entries[0].Message)
	}
	fields := entries[0].ContextMap()
	assertField(t, fields, "topic", string(config.TelemetryEventsTopic))
	if _, ok := fields["error"]; !ok {
		t.Error("expected error field")
	}
}

func assertField(t *testing.T, fields map[string]any, key, want string) {
	t.Helper()
	got, ok := fields[key]
	if !ok || got != want {
		t.Errorf("%s: got %#v, want %q", key, fields[key], want)
	}
}

func TestRecordTopic(t *testing.T) {
	if got := NewRecord(EventAuctionCompleted, Envelope{}).Topic(); got != config.TelemetryEventsTopic {
		t.Errorf("Topic(): got %q, want %q", got, config.TelemetryEventsTopic)
	}
}
