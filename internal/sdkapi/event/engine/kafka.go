package engine

import (
	"context"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/bidon-io/bidon-backend/config"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
)

type Kafka struct {
	Topics map[config.Topic]string
	Client *kgo.Client
}

func (e *Kafka) Produce(message event.LogMessage, handleErr func(error)) {
	e.TryProduce(message, func(err error) {
		if err != nil {
			handleErr(err)
		}
	})
}

// TryProduce enqueues a record without blocking the caller. done runs exactly
// once: nil after the broker accepts the record, or an error if the record
// cannot be built or the client buffer is full (an error wrapping
// kgo.ErrMaxBuffered).
func (e *Kafka) TryProduce(message event.LogMessage, done func(error)) {
	record, err := e.record(message)
	if err != nil {
		done(err)
		return
	}
	e.Client.TryProduce(context.Background(), record, func(_ *kgo.Record, err error) {
		if err != nil {
			err = fmt.Errorf("kafka produce record: %w", err)
		}
		done(err)
	})
}

func (e *Kafka) record(message event.LogMessage) (*kgo.Record, error) {
	topicStr := e.Topics[message.Topic]
	if topicStr == "" {
		return nil, fmt.Errorf("topic for %q not set", message.Topic)
	}
	return &kgo.Record{
		Topic:   topicStr,
		Value:   message.Value,
		Headers: kafkaHeaders(message.Headers),
	}, nil
}

func kafkaHeaders(headers map[string]string) []kgo.RecordHeader {
	if len(headers) == 0 {
		return nil
	}
	out := make([]kgo.RecordHeader, 0, len(headers))
	for key, value := range headers {
		out = append(out, kgo.RecordHeader{Key: key, Value: []byte(value)})
	}
	return out
}

func (e *Kafka) Ping(ctx context.Context) error {
	if err := e.Client.Ping(ctx); err != nil {
		return fmt.Errorf("kafka ping: %v", err)
	}
	return nil
}
