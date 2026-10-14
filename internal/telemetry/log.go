package telemetry

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
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
	fields := []zap.Field{
		zap.String("topic", string(message.Topic)),
		zap.String(HeaderEventName, message.Headers[HeaderEventName]),
	}
	msg, err := DecodeMessage(message)
	if err != nil {
		e.log().Debug("produce telemetry", append(fields, zap.Error(err))...)
		return
	}
	event, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(msg)
	if err != nil {
		e.log().Debug("produce telemetry", append(fields, zap.Error(err))...)
		return
	}
	e.log().Debug("produce telemetry", append(fields, zap.ByteString("event", event))...)
}

func (e *Log) Ping(_ context.Context) error {
	return nil
}
