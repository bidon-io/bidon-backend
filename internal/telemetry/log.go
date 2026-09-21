package telemetry

import (
	"context"

	"go.uber.org/zap"
)

type Log struct {
	Logger *zap.Logger
}

func (e *Log) log() *zap.Logger {
	if e == nil || e.Logger == nil {
		return zap.NewNop()
	}
	return e.Logger
}

func (e *Log) Produce(message LogMessage, _ func(error)) {
	rec, err := DecodeRecord(message)
	if err != nil {
		e.log().Debug("produce telemetry",
			zap.String("topic", string(message.Topic)),
			zap.Error(err),
		)
		return
	}
	e.log().Debug("produce telemetry", eventLogFields(rec)...)
}

func (e *Log) Ping(_ context.Context) error {
	return nil
}
