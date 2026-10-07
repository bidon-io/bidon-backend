package telemetry

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	telemetryv1 "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/telemetry/v1"
	telemetryproto "github.com/bidon-io/bidon-backend/schemas/proto"
)

type fakeRegistry struct {
	id         int
	err        error
	failFirst  int32
	calls      atomic.Int32
	subjects   []string
	lastSchema string
}

func (f *fakeRegistry) Register(_ context.Context, subject, schema string) (int, error) {
	if f.calls.Add(1) <= f.failFirst {
		return 0, errors.New("registry unavailable")
	}
	f.subjects = append(f.subjects, subject)
	f.lastSchema = schema
	return f.id, f.err
}

func registeredSerde(t *testing.T, reg *fakeRegistry) *confluentSerde {
	t.Helper()
	serde := newConfluentSerde(reg, "telemetry-events", nil)
	serde.registerAll(context.Background())
	return serde
}

func TestConfluentFrameRoundTrip(t *testing.T) {
	msg := &telemetryv1.DspResponseReceived{
		Envelope: &telemetryv1.Envelope{EventName: "dsp_response_received", AuctionId: "auc-1"},
		Dsp:      "bidmachine",
	}
	payload, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	serde := registeredSerde(t, &fakeRegistry{id: 7})
	framed := serde.frame(msg, payload)
	if bytes.Equal(framed, payload) {
		t.Fatal("expected Confluent prefix")
	}
	if framed[0] != 0 {
		t.Fatalf("magic byte: got %d, want 0", framed[0])
	}
	if !strings.Contains(telemetryproto.EventsProto, "org.bidon.telemetry.v1") {
		t.Fatal("embedded proto schema missing package")
	}
	if got := schemaSubject("telemetry-events", msg); got != "telemetry-events-org.bidon.telemetry.v1.DspResponseReceived" {
		t.Fatalf("subject: %q", got)
	}
	if idx := protoMessageIndex(msg); len(idx) != 1 || idx[0] == 0 {
		t.Fatalf("DspResponseReceived must not be proto file index 0, got %v", idx)
	}

	stripped := stripConfluentPrefix(framed)
	if !bytes.Equal(stripped, payload) {
		t.Fatal("stripConfluentPrefix must restore proto payload")
	}

	decoded, err := DecodeMessage(LogMessage{
		Value:   framed,
		Headers: protoHeaders(EventName(msg), msg),
	})
	if err != nil {
		t.Fatalf("DecodeMessage framed: %v", err)
	}
	if !proto.Equal(decoded, msg) {
		t.Fatalf("decoded = %v, want %v", decoded, msg)
	}
}

func TestConfluentSerdeFailOpen(t *testing.T) {
	msg := &telemetryv1.AuctionRequestReceived{Envelope: &telemetryv1.Envelope{EventName: "auction_request_received"}}
	payload, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	reg := &fakeRegistry{id: 3}
	serde := newConfluentSerde(reg, "telemetry-events", nil)
	framed := serde.frame(msg, payload)
	if !bytes.Equal(framed, payload) {
		t.Fatal("an unregistered subject must fail open to raw proto")
	}
	if got := reg.calls.Load(); got != 0 {
		t.Fatalf("frame must never call the registry, got %d calls", got)
	}
}

func TestConfluentSerdeRegisterAllRetries(t *testing.T) {
	msg := &telemetryv1.DspRequestSent{Envelope: &telemetryv1.Envelope{EventName: "dsp_request_sent"}}
	payload, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	reg := &fakeRegistry{id: 3, failFirst: 7}
	serde := newConfluentSerde(reg, "telemetry-events", nil)
	serde.retryMin = time.Millisecond
	serde.registerAll(context.Background())

	n := len(catalogTypes())
	if n != 5 {
		t.Fatalf("catalog types: got %d, want 5", n)
	}
	if got := len(reg.subjects); got != n {
		t.Fatalf("registered subjects: got %d, want %d", got, n)
	}
	if framed := serde.frame(msg, payload); framed[0] != 0 {
		t.Fatal("frame must use the id registered after retries")
	}
	_ = serde.frame(msg, payload)
	if got := reg.calls.Load(); got != int32(n)+reg.failFirst {
		t.Fatalf("Register calls: got %d, want %d", got, int32(n)+reg.failFirst)
	}
}

func TestConfluentSerdeRegisterAllStopsOnCancel(t *testing.T) {
	reg := &fakeRegistry{err: errors.New("registry down")}
	serde := newConfluentSerde(reg, "telemetry-events", nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		serde.registerAll(ctx)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("registerAll must return once ctx is done")
	}
}

func TestStripConfluentPrefixRawProtoUnchanged(t *testing.T) {
	raw := []byte{0x0a, 0x04, 0x74, 0x65, 0x73, 0x74}
	if got := stripConfluentPrefix(raw); !bytes.Equal(got, raw) {
		t.Fatal("raw proto must pass through")
	}
}

func TestEventEmitFramesWhenRegistrySet(t *testing.T) {
	engine := &recordingEngine{}
	logger := New(engine, nil)
	logger.event.serde = registeredSerde(t, &fakeRegistry{id: 11})

	logger.Event().DSPResponseReceived(testAuctionParams(), &adapters.DemandResponse{
		DemandID: adapter.BidmachineKey,
		Status:   http.StatusOK,
		Bid:      &adapters.DemandBid{Price: 1.23},
		StartTS:  0,
		EndTS:    15,
	})
	if len(engine.messages) != 1 {
		t.Fatalf("messages: %d", len(engine.messages))
	}
	msg := engine.messages[0]
	if msg.Value[0] != 0 {
		t.Fatal("produced value must be Confluent-framed")
	}
	decoded, err := DecodeMessage(msg)
	if err != nil {
		t.Fatalf("DecodeMessage: %v", err)
	}
	pb, ok := decoded.(*telemetryv1.DspResponseReceived)
	if !ok {
		t.Fatalf("decoded type = %T", decoded)
	}
	if pb.GetEnvelope().GetEventName() != "dsp_response_received" {
		t.Fatalf("event_name: %q", pb.GetEnvelope().GetEventName())
	}
}
