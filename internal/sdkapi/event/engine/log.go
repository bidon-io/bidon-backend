package engine

import (
	"context"
	"log"

	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
)

type Log struct{}

func (e *Log) Produce(ctx context.Context, topic event.Topic, message []byte, errorCb func(error)) error {
	log.Printf("PRODUCE EVENT %v: %s", topic, message)
	return nil
}
