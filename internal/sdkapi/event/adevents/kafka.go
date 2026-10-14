package adevents

import (
	"context"

	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event/engine"
)

// Kafka produces ad events through the shared sdkapi Kafka client.
// Enabled is off unless KAFKA_PRODUCE_AD_EVENTS=true; Produce then drops the record.
// Notification and telemetry records wrap this same client and share its buffer.
type Kafka struct {
	Producer *engine.Kafka
	Enabled  bool
}

func (k *Kafka) Produce(message event.LogMessage, handleErr func(error)) {
	if !k.Enabled {
		return
	}
	k.Producer.Produce(message, handleErr)
}

func (k *Kafka) Ping(ctx context.Context) error {
	if k.Producer == nil {
		return nil
	}
	return k.Producer.Ping(ctx)
}
