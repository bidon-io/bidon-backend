package telemetry

import (
	"context"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/bidon-io/bidon-backend/config"
)

type Kafka struct {
	Topics map[config.Topic]string
	Client *kgo.Client
}

func (e *Kafka) Produce(message LogMessage, handleErr func(error)) {
	topic := message.Topic
	topicStr := e.Topics[topic]
	if topicStr == "" {
		if handleErr != nil {
			handleErr(fmt.Errorf("topic for %q not set", topic))
		}
		return
	}

	record := &kgo.Record{
		Topic: topicStr,
		Value: message.Value,
	}
	e.Client.Produce(context.Background(), record, func(_ *kgo.Record, err error) {
		if err != nil && handleErr != nil {
			handleErr(fmt.Errorf("kafka produce record: %v", err))
		}
	})
}

func (e *Kafka) Ping(ctx context.Context) error {
	if err := e.Client.Ping(ctx); err != nil {
		return fmt.Errorf("kafka ping: %v", err)
	}
	return nil
}
