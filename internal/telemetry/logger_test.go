package telemetry

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/protobuf/proto"

	"github.com/bidon-io/bidon-backend/config"
	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/sdkapi"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
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

	rec, err := DecodeRecord(msg)
	if err != nil {
		t.Fatalf("DecodeRecord: %v", err)
	}

	if rec.EventName != EventDSPResponseReceived {
		t.Errorf("event_name: got %q", rec.EventName)
	}
	if rec.SchemaVersion != SchemaVersion {
		t.Errorf("schema_version: got %q", rec.SchemaVersion)
	}
	if rec.AuctionID != "auc-1" || rec.SessionID != "sess-1" {
		t.Errorf("ids: auction=%q session=%q", rec.AuctionID, rec.SessionID)
	}
	if rec.AdType != string(ad.BannerType) || rec.AdFormat != string(ad.BannerFormat) {
		t.Errorf("ad: type=%q format=%q", rec.AdType, rec.AdFormat)
	}
	if rec.Country != "US" || rec.TraceID != "trace-1" {
		t.Errorf("country/trace: %q %q", rec.Country, rec.TraceID)
	}
	if rec.Scope != ScopeBiddingRound {
		t.Errorf("scope: got %q", rec.Scope)
	}
	if rec.DSP != string(adapter.BidmachineKey) {
		t.Errorf("dsp: got %q", rec.DSP)
	}
	if rec.Outcome != OutcomeBid {
		t.Errorf("outcome: got %q", rec.Outcome)
	}
	if rec.AppID != 42 || rec.SamplingRate != 1.0 {
		t.Errorf("app_id=%d sampling_rate=%v", rec.AppID, rec.SamplingRate)
	}
	if rec.HTTPStatus != 200 || rec.LatencyMS != 15 || rec.Price != 1.23 {
		t.Errorf("http=%d latency=%d price=%v", rec.HTTPStatus, rec.LatencyMS, rec.Price)
	}
	if rec.PriceFloor != 0 {
		t.Errorf("dsp_response_received must not set price_floor, got %v", rec.PriceFloor)
	}
	if rec.EventID == "" {
		t.Error("event_id must be a non-empty UUID")
	}
	if rec.EventTS <= 0 {
		t.Errorf("event_ts: got %d, want unix ms", rec.EventTS)
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
		Value: []byte{0x00},
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
		t.Error("raw blob must not be logged; use named fields")
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

	unknown, ok := NewMessage("not_a_catalog_event")
	if ok || unknown != nil {
		t.Fatal("unknown event_name must not resolve")
	}
}

func assertField(t *testing.T, fields map[string]any, key, want string) {
	t.Helper()
	got, ok := fields[key]
	if !ok || got != want {
		t.Errorf("%s: got %#v, want %q", key, fields[key], want)
	}
}
