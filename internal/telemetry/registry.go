package telemetry

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	telemetryv1 "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/telemetry/v1"
)

const (
	HeaderEventName   = "event_name"
	HeaderMessageType = "protobuf_message"
)

// catalogEvent is a telemetry-events message: it embeds the Envelope and
// declares its event_name with the (event_name) option in events.proto.
type catalogEvent interface {
	proto.Message
	GetEnvelope() *telemetryv1.Envelope
}

// EventName returns the (event_name) option declared on msg's type, or "".
func EventName(msg proto.Message) string {
	name, _ := proto.GetExtension(msg.ProtoReflect().Descriptor().Options(), telemetryv1.E_EventName).(string)
	return name
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

// DecodeMessage decodes a produced telemetry-events value into the message
// type named by its protobuf_message header. Framed and raw values both work.
func DecodeMessage(msg LogMessage) (proto.Message, error) {
	name := msg.Headers[HeaderMessageType]
	if name == "" {
		return nil, fmt.Errorf("missing %s header", HeaderMessageType)
	}
	mt, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(name))
	if err != nil {
		return nil, fmt.Errorf("unknown %s %q: %w", HeaderMessageType, name, err)
	}
	out := mt.New().Interface()
	if err := proto.Unmarshal(stripConfluentPrefix(msg.Value), out); err != nil {
		return nil, err
	}
	return out, nil
}
