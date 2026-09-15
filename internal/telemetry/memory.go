package telemetry

import (
	"context"
	"encoding/json"
	"sync"

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

func (e *MemoryEngine) Records() []Record {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]Record, 0, len(e.Messages))
	for _, msg := range e.Messages {
		if msg.Topic != config.TelemetryEventsTopic {
			continue
		}
		var rec Record
		if err := json.Unmarshal(msg.Value, &rec); err != nil {
			continue
		}
		out = append(out, rec)
	}
	return out
}
