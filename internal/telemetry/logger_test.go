package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/protobuf/proto"

	"github.com/bidon-io/bidon-backend/config"
	telemetryv1 "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/telemetry/v1"
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
	if msg.Headers[HeaderEventName] != EventDSPResponseReceived {
		t.Errorf("event_name header: got %q", msg.Headers[HeaderEventName])
	}
	if msg.Headers[HeaderMessageType] != "org.bidon.telemetry.v1.DspResponseReceived" {
		t.Errorf("protobuf_message header: got %q", msg.Headers[HeaderMessageType])
	}
	if json.Valid(msg.Value) {
		t.Fatal("produced value must be protobuf, not JSON")
	}

	var pb telemetryv1.DspResponseReceived
	if err := proto.Unmarshal(msg.Value, &pb); err != nil {
		t.Fatalf("unmarshal proto: %v", err)
	}
	if pb.GetEnvelope() == nil {
		t.Fatal("envelope must be embedded")
	}

	got, err := DecodeRecord(msg)
	if err != nil {
		t.Fatalf("DecodeRecord: %v", err)
	}
	if got.EventName != EventDSPResponseReceived {
		t.Errorf("event_name: got %q", got.EventName)
	}
	if got.SchemaVersion != SchemaVersion {
		t.Errorf("schema_version: got %q", got.SchemaVersion)
	}
	if got.AuctionID != "auc-1" || got.SessionID != "sess-1" {
		t.Errorf("ids: auction=%q session=%q", got.AuctionID, got.SessionID)
	}
	if got.AdType != "banner" || got.AdFormat != "BANNER" {
		t.Errorf("ad: type=%q format=%q", got.AdType, got.AdFormat)
	}
	if got.Country != "US" || got.TraceID != "trace-1" {
		t.Errorf("country/trace: %q %q", got.Country, got.TraceID)
	}
	if got.Scope != "bidding_round" || got.DSP != "bidmachine" || got.Outcome != "bid" {
		t.Errorf("dsp fields: scope=%q dsp=%q outcome=%q", got.Scope, got.DSP, got.Outcome)
	}
	if got.AppID != 42 || got.SamplingRate != 1.0 {
		t.Errorf("app_id=%d sampling_rate=%v", got.AppID, got.SamplingRate)
	}
	if got.HTTPStatus != 200 || got.LatencyMS != 15 || got.Price != 1.23 {
		t.Errorf("http=%d latency=%d price=%v", got.HTTPStatus, got.LatencyMS, got.Price)
	}
	if got.EventID == "" {
		t.Error("event_id must be a non-empty UUID")
	}
	if got.EventTS <= 0 {
		t.Errorf("event_ts: got %d, want unix ms", got.EventTS)
	}
}

func TestCatalogEventRegistry(t *testing.T) {
	for _, name := range CatalogEventNames() {
		msg, ok := NewMessage(name)
		if !ok {
			t.Errorf("NewMessage(%q) missing from proto registry", name)
			continue
		}
		if messageTypeName(msg) == "" {
			t.Errorf("%s proto full name is empty", name)
		}
	}
	if _, ok := NewMessage("not_a_catalog_event"); ok {
		t.Fatal("unknown event_name must not resolve")
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
