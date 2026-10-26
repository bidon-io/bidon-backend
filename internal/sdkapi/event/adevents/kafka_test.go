package adevents

import (
	"errors"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/bidon-io/bidon-backend/config"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event/engine"
)

func unreachableProducer(t *testing.T, opts ...kgo.Opt) *engine.Kafka {
	t.Helper()
	client, err := kgo.NewClient(append([]kgo.Opt{kgo.SeedBrokers("127.0.0.1:1")}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Close)
	return &engine.Kafka{
		Client: client,
		Topics: map[config.Topic]string{config.AdEventsTopic: "ad-events"},
	}
}

func TestProduceSkipsAdEventsUnlessEnabled(t *testing.T) {
	producer := unreachableProducer(t, kgo.MaxBufferedRecords(1))
	k := &Kafka{Producer: producer}

	for i := 0; i < 5; i++ {
		k.Produce(event.LogMessage{Topic: config.AdEventsTopic, Value: []byte(`{}`)}, func(err error) {
			t.Errorf("disabled ad event produce: %v", err)
		})
	}

	errs := make(chan error, 2)
	msg := event.LogMessage{Topic: config.AdEventsTopic, Value: []byte(`{}`)}
	producer.Produce(msg, func(err error) { errs <- err })
	producer.Produce(msg, func(err error) { errs <- err })

	select {
	case err := <-errs:
		if !errors.Is(err, kgo.ErrMaxBuffered) {
			t.Fatalf("err = %v, want ErrMaxBuffered", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for buffer-full error; disabled ad events may have filled the buffer")
	}
	select {
	case err := <-errs:
		t.Fatalf("unexpected second error: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestProduceAdEventsWhenEnabled(t *testing.T) {
	k := &Kafka{
		Producer: unreachableProducer(t, kgo.MaxBufferedRecords(1)),
		Enabled:  true,
	}

	errs := make(chan error, 2)
	msg := event.LogMessage{Topic: config.AdEventsTopic, Value: []byte(`{}`)}
	k.Produce(msg, func(err error) { errs <- err })
	k.Produce(msg, func(err error) { errs <- err })

	select {
	case err := <-errs:
		if !errors.Is(err, kgo.ErrMaxBuffered) {
			t.Fatalf("err = %v, want ErrMaxBuffered", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for buffer-full error")
	}
}

func TestProduceDisabledDoesNotRequireClient(t *testing.T) {
	k := &Kafka{}
	k.Produce(event.LogMessage{Topic: config.AdEventsTopic, Value: []byte(`{}`)}, func(err error) {
		t.Errorf("disabled ad event produce: %v", err)
	})
}
