package telemetry

import (
	"context"
	"errors"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event/engine"
)

// ErrBufferFull means a telemetry record was dropped rather than wait for
// Kafka: the shared client buffer is full.
var ErrBufferFull = errors.New("telemetry buffer full")

// Kafka produces telemetry through the sdkapi event producer, so both share
// one produce path and kgo client. It never blocks: when the client buffer is
// full, TryProduce drops the record with ErrBufferFull instead of waiting.
// There is no separate telemetry quota; the client's MaxBufferedRecords
// (10,000 by default, shared with ad-events) is the only limit.
type Kafka struct {
	Producer *engine.Kafka
}

func (e *Kafka) Produce(message LogMessage, handleErr func(error)) {
	if handleErr == nil {
		handleErr = func(error) {}
	}
	e.Producer.TryProduce(event.LogMessage{
		Topic:   message.Topic,
		Value:   message.Value,
		Headers: message.Headers,
	}, func(err error) {
		if errors.Is(err, kgo.ErrMaxBuffered) {
			err = ErrBufferFull
		}
		if err != nil {
			handleErr(err)
		}
	})
}

func (e *Kafka) Ping(ctx context.Context) error {
	return e.Producer.Ping(ctx)
}
