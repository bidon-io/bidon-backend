package telemetry

import (
	"context"
	"log"
)

type Log struct{}

func (e *Log) Produce(message LogMessage, _ func(error)) {
	log.Printf("PRODUCE TELEMETRY %T(%v): %s", message.Topic, message.Topic, message.Value)
}

func (e *Log) Ping(_ context.Context) error {
	return nil
}
