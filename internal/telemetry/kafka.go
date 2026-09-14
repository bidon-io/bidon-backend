package telemetry

import (
	"context"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"

	"github.com/bidon-io/bidon-backend/config"
)

type Kafka struct {
	Topics map[config.Topic]string
	Client *kgo.Client
	Logger *zap.Logger
}

func (e *Kafka) log() *zap.Logger {
	if e == nil || e.Logger == nil {
		return zap.NewNop()
	}
	return e.Logger
}

func (e *Kafka) Produce(message LogMessage, handleErr func(error)) {
	topic := message.Topic
	topicStr := e.Topics[topic]
	if topicStr == "" {
		err := fmt.Errorf("topic for %q not set", topic)
		e.log().Error("telemetry topic not set",
			zap.String("topic", string(topic)),
			zap.Error(err),
		)
		if handleErr != nil {
			handleErr(err)
		}
		return
	}

	record := &kgo.Record{
		Topic: topicStr,
		Value: message.Value,
	}
	e.Client.Produce(context.Background(), record, func(_ *kgo.Record, err error) {
		if err == nil {
			return
		}
		e.log().Error("kafka produce record",
			zap.String("topic", topicStr),
			zap.Error(err),
		)
		if handleErr != nil {
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
