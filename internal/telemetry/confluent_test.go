package telemetry

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	telemetryv1 "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/telemetry/v1"
	telemetryproto "github.com/bidon-io/bidon-backend/schemas/proto"
)

type fakeRegistry struct {
	id         int
	err        error
	calls      atomic.Int32
	subjects   []string
	lastSchema string
}

func (f *fakeRegistry) Register(_ context.Context, subject, schema string) (int, error) {
	f.calls.Add(1)
	f.subjects = append(f.subjects, subject)
	f.lastSchema = schema
	return f.id, f.err
}

func TestConfluentFrameRoundTrip(t *testing.T) {
	msg := &telemetryv1.DspResponseReceived{
		Envelope: &telemetryv1.Envelope{EventName: EventDSPResponseReceived, AuctionId: "auc-1"},
		Dsp:      "bidmachine",
	}
	payload, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	reg := &fakeRegistry{id: 7}
	serde := newConfluentSerde(reg, "telemetry-events", nil)
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

	rec, err := DecodeRecord(LogMessage{
		Value:   framed,
		Headers: protoHeaders(EventDSPResponseReceived, msg),
	})
	if err != nil {
		t.Fatalf("DecodeRecord framed: %v", err)
	}
	if rec.AuctionID != "auc-1" || rec.DSP != "bidmachine" {
		t.Fatalf("decoded record: %+v", rec)
	}
}

func TestConfluentSerdeFailOpen(t *testing.T) {
	msg := &telemetryv1.AuctionRequestReceived{Envelope: &telemetryv1.Envelope{EventName: EventAuctionRequestReceived}}
	payload, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	reg := &fakeRegistry{err: errors.New("registry down")}
	framed := newConfluentSerde(reg, "telemetry-events", nil).frame(msg, payload)
	if !bytes.Equal(framed, payload) {
		t.Fatal("registry errors must fail open to raw proto")
	}
}

func TestConfluentSerdeCachesID(t *testing.T) {
	msg := &telemetryv1.DspRequestSent{Envelope: &telemetryv1.Envelope{EventName: EventDSPRequestSent}}
	payload, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	reg := &fakeRegistry{id: 3}
	serde := newConfluentSerde(reg, "telemetry-events", nil)
	_ = serde.frame(msg, payload)
	_ = serde.frame(msg, payload)
	if got := reg.calls.Load(); got != 1 {
		t.Fatalf("Register calls: got %d, want 1", got)
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
	logger.Event.serde = newConfluentSerde(&fakeRegistry{id: 11}, "telemetry-events", nil)

	logger.Event.DSPResponseReceived(testAuctionParams(), &adapters.DemandResponse{
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
	rec, err := DecodeRecord(msg)
	if err != nil {
		t.Fatalf("DecodeRecord: %v", err)
	}
	if rec.EventName != EventDSPResponseReceived {
		t.Fatalf("event_name: %q", rec.EventName)
	}
}
