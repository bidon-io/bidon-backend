package telemetry

import (
	"context"
	"sync"

	"google.golang.org/protobuf/proto"

	"github.com/bidon-io/bidon-backend/config"
)

// MemoryEngine records produced events for tests.
type MemoryEngine struct {
	mu       sync.Mutex
	Messages []LogMessage
}

func (e *MemoryEngine) Produce(message LogMessage, _ func(error)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Messages = append(e.Messages, message)
}

func (e *MemoryEngine) Ping(_ context.Context) error { return nil }

// Events decodes every telemetry-events message produced so far, in order.
func (e *MemoryEngine) Events() []proto.Message {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]proto.Message, 0, len(e.Messages))
	for _, msg := range e.Messages {
		if msg.Topic != config.TelemetryEventsTopic {
			continue
		}
		decoded, err := DecodeMessage(msg)
		if err != nil {
			continue
		}
		out = append(out, decoded)
	}
	return out
}

// EventsOf returns the produced events of type T, in order.
func EventsOf[T proto.Message](e *MemoryEngine) []T {
	var out []T
	for _, msg := range e.Events() {
		if typed, ok := msg.(T); ok {
			out = append(out, typed)
		}
	}
	return out
}
