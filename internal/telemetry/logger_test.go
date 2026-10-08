package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/bidon-io/bidon-backend/config"
	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/sdkapi"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event/engine"
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

	logger.Event().DSPResponseReceived(testAuctionParams(), &adapters.DemandResponse{
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
	if msg.Headers[HeaderEventName] != "dsp_response_received" {
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

	env := pb.GetEnvelope()
	if env.GetEventName() != "dsp_response_received" {
		t.Errorf("event_name: got %q", env.GetEventName())
	}
	if env.GetSchemaVersion() != SchemaVersion {
		t.Errorf("schema_version: got %q", env.GetSchemaVersion())
	}
	if env.GetAuctionId() != "auc-1" || env.GetSessionId() != "sess-1" {
		t.Errorf("ids: auction=%q session=%q", env.GetAuctionId(), env.GetSessionId())
	}
	if env.GetAdType() != string(ad.BannerType) || env.GetAdFormat() != string(ad.BannerFormat) {
		t.Errorf("ad: type=%q format=%q", env.GetAdType(), env.GetAdFormat())
	}
	if env.GetCountry() != "US" || env.GetTraceId() != "trace-1" {
		t.Errorf("country/trace: %q %q", env.GetCountry(), env.GetTraceId())
	}
	if env.GetAppId() != 42 || env.GetSamplingRate() != 1.0 {
		t.Errorf("app_id=%d sampling_rate=%v", env.GetAppId(), env.GetSamplingRate())
	}
	if env.GetEventId() == "" {
		t.Error("event_id must be a non-empty UUID")
	}
	if env.GetEventTs() <= 0 {
		t.Errorf("event_ts: got %d, want unix ms", env.GetEventTs())
	}
	if pb.GetScope() != telemetryv1.Scope_SCOPE_BIDDING_ROUND {
		t.Errorf("scope: got %v", pb.GetScope())
	}
	if pb.GetDsp() != string(adapter.BidmachineKey) {
		t.Errorf("dsp: got %q", pb.GetDsp())
	}
	if pb.GetOutcome() != telemetryv1.Outcome_OUTCOME_BID {
		t.Errorf("outcome: got %v", pb.GetOutcome())
	}
	if pb.GetHttpStatus() != 200 || pb.GetLatencyMs() != 15 || pb.GetPrice() != 1.23 {
		t.Errorf("http=%d latency=%d price=%v", pb.GetHttpStatus(), pb.GetLatencyMs(), pb.GetPrice())
	}

	decoded, err := DecodeMessage(msg)
	if err != nil {
		t.Fatalf("DecodeMessage: %v", err)
	}
	if !proto.Equal(decoded, &pb) {
		t.Errorf("DecodeMessage = %v, want %v", decoded, &pb)
	}
}

func TestLoggerNilSafe(t *testing.T) {
	params := testAuctionParams()

	Event{}.AuctionRequestReceived(params)
	New(nil, nil).Event().AuctionRequestReceived(params)
}

func TestKafkaProduceEmptyTopic(t *testing.T) {
	producer := &Kafka{Producer: &engine.Kafka{Topics: map[config.Topic]string{}}}
	called := false

	producer.Produce(LogMessage{
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
	logger := New(&Kafka{Producer: &engine.Kafka{Topics: map[config.Topic]string{}}}, zap.New(core))

	logger.Event().AuctionRequestReceived(Params{
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
	assertField(t, fields, "event_name", "auction_request_received")
	assertField(t, fields, "auction_id", "auc-structured")
	assertField(t, fields, "session_id", "sess-structured")
	if fields["app_id"] != int64(7) {
		t.Errorf("app_id: got %#v, want 7", fields["app_id"])
	}
	if _, ok := fields["error"]; !ok {
		t.Error("expected error field")
	}
	for _, key := range []string{"dsp", "trace_id"} {
		if _, ok := fields[key]; ok {
			t.Errorf("%s: empty value must be omitted", key)
		}
	}
}

func TestAttrsLogFieldsOmitsEmpty(t *testing.T) {
	enc := zapcore.NewMapObjectEncoder()
	for _, f := range attrsLogFields("", "", attrs{}) {
		f.AddTo(enc)
	}
	if len(enc.Fields) != 1 || enc.Fields["topic"] != string(config.TelemetryEventsTopic) {
		t.Errorf("fields: got %v, want only topic", enc.Fields)
	}
}

func TestLogEngineStructured(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	engine := &Log{Logger: zap.New(core)}
	logger := New(engine, nil)

	logger.Event().DSPResponseReceived(Params{
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
	assertField(t, fields, "event_name", "dsp_response_received")
	raw, ok := fields["event"].(string)
	if !ok {
		t.Fatalf("event field: got %#v, want protojson string", fields["event"])
	}
	var event struct {
		Envelope struct {
			AuctionID string `json:"auction_id"`
			TraceID   string `json:"trace_id"`
		} `json:"envelope"`
		DSP string `json:"dsp"`
	}
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		t.Fatalf("event is not JSON: %v (%s)", err, raw)
	}
	if event.Envelope.AuctionID != "auc-log" || event.Envelope.TraceID != "trace-log" || event.DSP != string(adapter.BidmachineKey) {
		t.Errorf("event = %s", raw)
	}
}

// TestCatalogEventNames pins every catalog message to its declared
// (event_name). A new message with an envelope must be added here.
func TestCatalogEventNames(t *testing.T) {
	want := map[string]string{
		"org.bidon.telemetry.v1.AuctionRequestReceived": "auction_request_received",
		"org.bidon.telemetry.v1.AuctionCompleted":       "auction_completed",
		"org.bidon.telemetry.v1.DspRequestSent":         "dsp_request_sent",
		"org.bidon.telemetry.v1.DspResponseReceived":    "dsp_response_received",
		"org.bidon.telemetry.v1.DspResponseRejected":    "dsp_response_rejected",
	}

	got := map[string]string{}
	seen := map[string]string{}
	msgs := telemetryv1.File_org_bidon_telemetry_v1_events_proto.Messages()
	for i := 0; i < msgs.Len(); i++ {
		d := msgs.Get(i)
		if d.Fields().ByName("envelope") == nil {
			continue
		}
		mt, err := protoregistry.GlobalTypes.FindMessageByName(d.FullName())
		if err != nil {
			t.Fatalf("%s not registered: %v", d.FullName(), err)
		}
		msg := mt.New().Interface()
		if _, ok := msg.(catalogEvent); !ok {
			t.Errorf("%s does not satisfy catalogEvent", d.FullName())
		}
		name := EventName(msg)
		if name == "" {
			t.Errorf("%s has an envelope but no (event_name) option", d.FullName())
			continue
		}
		if other, dup := seen[name]; dup {
			t.Errorf("event_name %q declared by both %s and %s", name, other, d.FullName())
		}
		seen[name] = string(d.FullName())
		got[string(d.FullName())] = name
	}

	for typ, name := range want {
		if got[typ] != name {
			t.Errorf("%s event_name = %q, want %q", typ, got[typ], name)
		}
	}
	for typ := range got {
		if _, ok := want[typ]; !ok {
			t.Errorf("%s is a catalog message missing from this test", typ)
		}
	}
	if EventName(&telemetryv1.Envelope{}) != "" {
		t.Error("Envelope is not an event and must not declare event_name")
	}
}

func TestEmitDropsMessageWithoutEnvelope(t *testing.T) {
	engine := &recordingEngine{}
	New(engine, nil).Event().emit(&telemetryv1.DspRequestSent{}, "", attrs{})
	if len(engine.messages) != 0 {
		t.Fatalf("emit without envelope produced %d messages", len(engine.messages))
	}
}

type failingEngine struct {
	err error
}

func (e failingEngine) Produce(_ LogMessage, handleErr func(error)) {
	handleErr(e.err)
}

func (e failingEngine) Ping(_ context.Context) error {
	return nil
}

func TestEmitCountsDrops(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		reason  string
		wantLog bool
	}{
		{name: "buffer full is counted, not logged", err: ErrBufferFull, reason: DropReasonBufferFull},
		{name: "produce error is counted and logged", err: errors.New("broker said no"), reason: DropReasonProduce, wantLog: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := TelemetryDroppedTotal.WithLabelValues(tt.reason)
			before := counterValue(t, counter)
			core, logs := observer.New(zap.ErrorLevel)

			New(failingEngine{err: tt.err}, zap.New(core)).Event().AuctionRequestReceived(testAuctionParams())

			if got := counterValue(t, counter) - before; got != 1 {
				t.Errorf("telemetry_dropped_total{reason=%q} delta = %v, want 1", tt.reason, got)
			}
			if gotLog := logs.Len() > 0; gotLog != tt.wantLog {
				t.Errorf("logged = %v, want %v", gotLog, tt.wantLog)
			}
		})
	}
}

func TestDecodeMessageErrors(t *testing.T) {
	if _, err := DecodeMessage(LogMessage{}); err == nil {
		t.Error("missing protobuf_message header must error")
	}
	if _, err := DecodeMessage(LogMessage{Headers: map[string]string{HeaderMessageType: "org.bidon.telemetry.v1.Nope"}}); err == nil {
		t.Error("unknown message type must error")
	}
}

func assertField(t *testing.T, fields map[string]any, key, want string) {
	t.Helper()
	got, ok := fields[key]
	if !ok || got != want {
		t.Errorf("%s: got %#v, want %q", key, fields[key], want)
	}
}
