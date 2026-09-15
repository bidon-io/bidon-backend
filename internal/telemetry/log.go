package telemetry

import (
	"context"
	"encoding/json"

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
	var rec Record
	if err := json.Unmarshal(message.Value, &rec); err != nil {
		e.log().Debug("produce telemetry",
			zap.String("topic", string(message.Topic)),
			zap.Error(err),
		)
		return
	}
	e.log().Debug("produce telemetry", envelopeLogFields(message.Topic, rec.Envelope, rec.DSP)...)
}

func (e *Log) Ping(_ context.Context) error {
	return nil
}
