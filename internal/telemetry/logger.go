package telemetry

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"

	"github.com/bidon-io/bidon-backend/config"
)

type Logger struct {
	Engine LoggerEngine
	Logger *zap.Logger
}

type LoggerEngine interface {
	Produce(message LogMessage, handleErr func(error))
	Ping(ctx context.Context) error
}

type LogMessage struct {
	Topic config.Topic
	Value []byte
}

func (l *Logger) log() *zap.Logger {
	if l == nil || l.Logger == nil {
		return zap.NewNop()
	}
	return l.Logger
}

func (l *Logger) Log(ev Event, handleErr func(error)) {
	if l == nil || l.Engine == nil {
		return
	}

	logger := l.log().With(eventLogFields(ev)...)

	message, err := json.Marshal(ev)
	if err != nil {
		logger.Error("marshal telemetry event", zap.Error(err))
		if handleErr != nil {
			handleErr(err)
		}
		return
	}

	l.Engine.Produce(LogMessage{Topic: ev.Topic(), Value: message}, func(err error) {
		logger.Error("produce telemetry event", zap.Error(err))
		if handleErr != nil {
			handleErr(err)
		}
	})
}

func eventLogFields(ev Event) []zap.Field {
	if ev == nil {
		return nil
	}

	fields := []zap.Field{zap.String("topic", string(ev.Topic()))}
	rec, ok := eventRecord(ev)
	if !ok {
		return fields
	}

	fields = append(fields,
		zap.String("event_name", rec.EventName),
		zap.String("event_id", rec.EventID),
		zap.String("auction_id", rec.AuctionID),
		zap.Int64("app_id", rec.AppID),
		zap.String("session_id", rec.SessionID),
	)
	if rec.DSP != "" {
		fields = append(fields, zap.String("dsp", rec.DSP))
	}
	if rec.TraceID != "" {
		fields = append(fields, zap.String("trace_id", rec.TraceID))
	}
	return fields
}

func eventRecord(ev Event) (Record, bool) {
	switch v := ev.(type) {
	case Record:
		return v, true
	case *Record:
		if v == nil {
			return Record{}, false
		}
		return *v, true
	default:
		return Record{}, false
	}
}
