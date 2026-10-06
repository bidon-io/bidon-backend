package telemetry

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/bidon-io/bidon-backend/config"
)

type LoggerEngine interface {
	Produce(message LogMessage, handleErr func(error))
	Ping(ctx context.Context) error
}

type LogMessage struct {
	Topic   config.Topic
	Value   []byte
	Headers map[string]string
}

// Logger is the sdkapi telemetry handle. Catalog emits go through Event().
type Logger struct {
	event Event
}

func New(engine LoggerEngine, log *zap.Logger) *Logger {
	return &Logger{event: Event{engine: engine, log: log}}
}

// Event returns the catalog emitter. A nil Logger yields a no-op Event.
func (l *Logger) Event() Event {
	if l == nil {
		return Event{}
	}
	return l.event
}

// Nop discards catalog events. Use it when no engine is configured.
var Nop = New(nil, nil)

// Event produces catalog rows onto telemetry-events.
type Event struct {
	engine LoggerEngine
	log    *zap.Logger
	serde  *confluentSerde
}

func (e Event) logf() *zap.Logger {
	if e.log == nil {
		return zap.NewNop()
	}
	return e.log
}

func (e Event) emit(msg catalogEvent, dsp string, a attrs) {
	if e.engine == nil {
		return
	}

	eventName := EventName(msg)

	env := msg.GetEnvelope()
	if eventName == "" || env == nil {
		e.logError("telemetry message missing (event_name) option or envelope", eventName, dsp, a,
			zap.String(HeaderMessageType, messageTypeName(msg)),
		)
		return
	}
	env.EventName = eventName

	message, err := proto.Marshal(msg)
	if err != nil {
		e.logError("marshal telemetry event", eventName, dsp, a, zap.Error(err))
		return
	}
	message = e.serde.frame(msg, message)

	e.engine.Produce(LogMessage{
		Topic:   config.TelemetryEventsTopic,
		Value:   message,
		Headers: protoHeaders(eventName, msg),
	}, func(err error) {
		e.logError("produce telemetry event", eventName, dsp, a, zap.Error(err))
	})
}

// logError builds the context fields only when there is something to log;
// emit runs per DSP per auction.
func (e Event) logError(msg, eventName, dsp string, a attrs, fields ...zap.Field) {
	e.logf().Error(msg, append(attrsLogFields(eventName, dsp, a), fields...)...)
}

func attrsLogFields(eventName, dsp string, a attrs) []zap.Field {
	fields := []zap.Field{zap.String("topic", string(config.TelemetryEventsTopic))}
	for _, f := range []struct{ key, value string }{
		{"event_name", eventName},
		{"auction_id", a.AuctionID},
		{"session_id", a.SessionID},
		{"dsp", dsp},
		{"trace_id", a.TraceID},
	} {
		if f.value != "" {
			fields = append(fields, zap.String(f.key, f.value))
		}
	}
	if a.AppID != 0 {
		fields = append(fields, zap.Int64("app_id", a.AppID))
	}
	return fields
}
