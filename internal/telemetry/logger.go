package telemetry

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

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
	Topic   config.Topic
	Value   []byte
	Headers map[string]string
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

	rec, ok := eventRecord(ev)
	if !ok {
		err := fmt.Errorf("telemetry event is not a catalog Record")
		logger.Error("marshal telemetry event", zap.Error(err))
		if handleErr != nil {
			handleErr(err)
		}
		return
	}

	msg, err := recordToProto(rec)
	if err != nil {
		logger.Error("marshal telemetry event", zap.Error(err))
		if handleErr != nil {
			handleErr(err)
		}
		return
	}

	// Raw protobuf. Confluent framing (magic + schema id) belongs here when
	// SCHEMA_REGISTRY_URL is set — see schemas/proto/org/bidon/telemetry/v1/CONFLUENT.md.
	message, err := proto.Marshal(msg)
	if err != nil {
		logger.Error("marshal telemetry event", zap.Error(err))
		if handleErr != nil {
			handleErr(err)
		}
		return
	}

	l.Engine.Produce(LogMessage{
		Topic:   ev.Topic(),
		Value:   message,
		Headers: protoHeaders(rec.EventName, msg),
	}, func(err error) {
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
