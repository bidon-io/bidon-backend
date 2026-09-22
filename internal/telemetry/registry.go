package telemetry

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	telemetryv1 "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/telemetry/v1"
)

const (
	HeaderEventName   = "event_name"
	HeaderMessageType = "protobuf_message"
)

// NewMessage returns an empty proto message for a catalog event_name.
func NewMessage(eventName string) (proto.Message, bool) {
	switch eventName {
	case EventAuctionRequestReceived:
		return &telemetryv1.AuctionRequestReceived{}, true
	case EventAuctionCompleted:
		return &telemetryv1.AuctionCompleted{}, true
	case EventDSPRequestSent:
		return &telemetryv1.DspRequestSent{}, true
	case EventDSPResponseReceived:
		return &telemetryv1.DspResponseReceived{}, true
	case EventDSPResponseRejected:
		return &telemetryv1.DspResponseRejected{}, true
	default:
		return nil, false
	}
}

// CatalogEventNames is the proto registry order for catalog emits.
func CatalogEventNames() []string {
	return []string{
		EventAuctionRequestReceived,
		EventAuctionCompleted,
		EventDSPRequestSent,
		EventDSPResponseReceived,
		EventDSPResponseRejected,
	}
}

func messageTypeName(msg proto.Message) string {
	return string(msg.ProtoReflect().Descriptor().FullName())
}

func protoHeaders(eventName string, msg proto.Message) map[string]string {
	return map[string]string{
		HeaderEventName:   eventName,
		HeaderMessageType: messageTypeName(msg),
	}
}

func DecodeRecord(msg LogMessage) (Record, error) {
	name := ""
	if msg.Headers != nil {
		name = msg.Headers[HeaderEventName]
	}
	if name == "" {
		return Record{}, fmt.Errorf("missing %s header", HeaderEventName)
	}
	protoMsg, ok := NewMessage(name)
	if !ok {
		return Record{}, fmt.Errorf("unknown event_name %q", name)
	}
	if err := proto.Unmarshal(stripConfluentPrefix(msg.Value), protoMsg); err != nil {
		return Record{}, err
	}
	return recordFromProto(protoMsg), nil
}
