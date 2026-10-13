package event

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bidon-io/bidon-backend/config"
)

type Logger struct {
	// Engine produces every topic. Ad events use AdEvents instead when it is set.
	Engine LoggerEngine
	// AdEvents produces ad events. Nil sends them through Engine.
	AdEvents LoggerEngine
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

func (l *Logger) Log(event Event, handleErr func(error)) {
	topic := event.Topic()

	message, err := json.Marshal(event)
	if err != nil {
		handleErr(fmt.Errorf("marshal %q event payload: %v", topic, err))
	}

	logMessage := LogMessage{
		Topic: topic,
		Value: message,
	}

	l.engine(topic).Produce(logMessage, func(err error) {
		handleErr(fmt.Errorf("produce %q message: %v", logMessage.Topic, err))
	})
}

func (l *Logger) engine(topic config.Topic) LoggerEngine {
	if topic == config.AdEventsTopic && l.AdEvents != nil {
		return l.AdEvents
	}
	return l.Engine
}
