package telemetry

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bidon-io/bidon-backend/config"
)

type Logger struct {
	Engine LoggerEngine
}

type LoggerEngine interface {
	Produce(message LogMessage, handleErr func(error))
	Ping(ctx context.Context) error
}

type LogMessage struct {
	Topic config.Topic
	Value []byte
}

func (l *Logger) Log(ev Event, handleErr func(error)) {
	if l == nil || l.Engine == nil {
		return
	}
	if handleErr == nil {
		handleErr = func(error) {}
	}

	topic := ev.Topic()

	message, err := json.Marshal(ev)
	if err != nil {
		handleErr(fmt.Errorf("marshal %q event payload: %v", topic, err))
		return
	}

	logMessage := LogMessage{
		Topic: topic,
		Value: message,
	}

	l.Engine.Produce(logMessage, func(err error) {
		handleErr(fmt.Errorf("produce %q message: %v", logMessage.Topic, err))
	})
}
