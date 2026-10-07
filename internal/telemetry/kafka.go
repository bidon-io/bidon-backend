package telemetry

import (
	"context"

	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event/engine"
)

// Kafka produces telemetry through the sdkapi event producer, so both share
// one produce path and kgo client.
type Kafka struct {
	Producer *engine.Kafka
}

func (e *Kafka) Produce(message LogMessage, handleErr func(error)) {
	if handleErr == nil {
		handleErr = func(error) {}
	}
	e.Producer.Produce(event.LogMessage{
		Topic:   message.Topic,
		Value:   message.Value,
		Headers: message.Headers,
	}, handleErr)
}

func (e *Kafka) Ping(ctx context.Context) error {
	return e.Producer.Ping(ctx)
}
