package telemetry

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event/engine"
)

// DefaultMaxBuffered is a quarter of kgo's default MaxBufferedRecords (10k),
// leaving the rest of the shared client buffer to ad-events.
const DefaultMaxBuffered = 2500

// ErrBufferFull means a telemetry record was dropped rather than wait for
// Kafka: telemetry hit MaxBuffered, or the shared client buffer is full.
var ErrBufferFull = errors.New("telemetry buffer full")

// Kafka produces telemetry through the sdkapi event producer, so both share
// one produce path and kgo client. It never blocks: when Kafka is slow or
// down, records are dropped with ErrBufferFull instead.
type Kafka struct {
	Producer *engine.Kafka
	// MaxBuffered caps telemetry records awaiting delivery, so telemetry
	// cannot fill the shared client buffer and block ad-events. Zero means
	// DefaultMaxBuffered.
	MaxBuffered int64

	buffered atomic.Int64
}

func (e *Kafka) Produce(message LogMessage, handleErr func(error)) {
	if handleErr == nil {
		handleErr = func(error) {}
	}
	if e.buffered.Add(1) > e.maxBuffered() {
		e.buffered.Add(-1)
		handleErr(ErrBufferFull)
		return
	}
	e.Producer.TryProduce(event.LogMessage{
		Topic:   message.Topic,
		Value:   message.Value,
		Headers: message.Headers,
	}, func(err error) {
		e.buffered.Add(-1)
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

func (e *Kafka) maxBuffered() int64 {
	if e.MaxBuffered > 0 {
		return e.MaxBuffered
	}
	return DefaultMaxBuffered
}
