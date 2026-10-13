package event

import (
	"context"
	"testing"

	"github.com/bidon-io/bidon-backend/config"
)

type stubEvent struct {
	topic config.Topic
}

func (e stubEvent) Topic() config.Topic { return e.topic }

type captureEngine struct {
	topics []config.Topic
}

func (e *captureEngine) Produce(message LogMessage, _ func(error)) {
	e.topics = append(e.topics, message.Topic)
}

func (e *captureEngine) Ping(context.Context) error { return nil }

func TestLoggerRoutesAdEvents(t *testing.T) {
	shared := &captureEngine{}
	ad := &captureEngine{}
	logger := &Logger{Engine: shared, AdEvents: ad}

	logger.Log(stubEvent{topic: config.AdEventsTopic}, func(error) {})
	logger.Log(stubEvent{topic: config.NotificationEventsTopic}, func(error) {})

	if len(ad.topics) != 1 || ad.topics[0] != config.AdEventsTopic {
		t.Fatalf("ad events engine: got %v", ad.topics)
	}
	if len(shared.topics) != 1 || shared.topics[0] != config.NotificationEventsTopic {
		t.Fatalf("shared engine: got %v", shared.topics)
	}
}

func TestLoggerUsesEngineWhenAdEventsUnset(t *testing.T) {
	shared := &captureEngine{}
	logger := &Logger{Engine: shared}

	logger.Log(stubEvent{topic: config.AdEventsTopic}, func(error) {})

	if len(shared.topics) != 1 || shared.topics[0] != config.AdEventsTopic {
		t.Fatalf("shared engine: got %v", shared.topics)
	}
}
