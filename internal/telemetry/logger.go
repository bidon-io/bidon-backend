package telemetry

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"

	"github.com/bidon-io/bidon-backend/config"
)

type LoggerEngine interface {
	Produce(message LogMessage, handleErr func(error))
	Ping(ctx context.Context) error
}

type LogMessage struct {
	Topic config.Topic
	Value []byte
}

// Logger is the sdkapi telemetry handle. Catalog emits go through Event.
type Logger struct {
	Event Event
}

func New(engine LoggerEngine, log *zap.Logger) *Logger {
	return &Logger{Event: Event{engine: engine, log: log}}
}

// Nop discards catalog events. Use it when no engine is configured.
var Nop = New(nil, nil)

// Event produces catalog rows onto telemetry-events.
type Event struct {
	engine LoggerEngine
	log    *zap.Logger
}

func (e Event) logf() *zap.Logger {
	if e.log == nil {
		return zap.NewNop()
	}
	return e.log
}

func (e Event) emit(eventName, dsp string, a attrs, v any) {
	if e.engine == nil {
		return
	}

	logger := e.logf().With(attrsLogFields(eventName, dsp, a)...)

	message, err := json.Marshal(v)
	if err != nil {
		logger.Error("marshal telemetry event", zap.Error(err))
		return
	}

	e.engine.Produce(LogMessage{Topic: config.TelemetryEventsTopic, Value: message}, func(err error) {
		logger.Error("produce telemetry event", zap.Error(err))
	})
}

func attrsLogFields(eventName, dsp string, a attrs) []zap.Field {
	fields := []zap.Field{
		zap.String("topic", string(config.TelemetryEventsTopic)),
		zap.String("event_name", eventName),
		zap.String("auction_id", a.AuctionID),
		zap.Int64("app_id", a.AppID),
		zap.String("session_id", a.SessionID),
	}
	if dsp != "" {
		fields = append(fields, zap.String("dsp", dsp))
	}
	if a.TraceID != "" {
		fields = append(fields, zap.String("trace_id", a.TraceID))
	}
	return fields
}

func envelopeLogFields(topic config.Topic, env Envelope, dsp string) []zap.Field {
	fields := []zap.Field{
		zap.String("topic", string(topic)),
		zap.String("event_name", env.EventName),
		zap.String("event_id", env.EventID),
		zap.String("auction_id", env.AuctionID),
		zap.Int64("app_id", env.AppID),
		zap.String("session_id", env.SessionID),
	}
	if dsp != "" {
		fields = append(fields, zap.String("dsp", dsp))
	}
	if env.TraceID != "" {
		fields = append(fields, zap.String("trace_id", env.TraceID))
	}
	return fields
}
