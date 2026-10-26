package engine

import (
	"errors"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/bidon-io/bidon-backend/config"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
)

func TestKafkaHeaders(t *testing.T) {
	if got := kafkaHeaders(nil); got != nil {
		t.Errorf("nil headers: got %v, want nil", got)
	}

	got := kafkaHeaders(map[string]string{"event_name": "dsp_request_sent"})
	want := []kgo.RecordHeader{{Key: "event_name", Value: []byte("dsp_request_sent")}}
	if len(got) != 1 || got[0].Key != want[0].Key || string(got[0].Value) != string(want[0].Value) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func unreachableClient(t *testing.T, opts ...kgo.Opt) *kgo.Client {
	t.Helper()
	client, err := kgo.NewClient(append([]kgo.Opt{kgo.SeedBrokers("127.0.0.1:1")}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Close)
	return client
}

func TestProduceDoesNotBlockWhenBufferFull(t *testing.T) {
	k := &Kafka{
		Client: unreachableClient(t, kgo.MaxBufferedRecords(1)),
		Topics: map[config.Topic]string{config.NotificationEventsTopic: "notification-events"},
	}

	const n = 3
	errs := make(chan error, n)
	returned := make(chan struct{})
	go func() {
		for i := 0; i < n; i++ {
			k.Produce(event.LogMessage{Topic: config.NotificationEventsTopic, Value: []byte(`{}`)}, func(err error) {
				errs <- err
			})
		}
		close(returned)
	}()
	select {
	case <-returned:
	case <-time.After(5 * time.Second):
		t.Fatal("Produce blocked while Kafka is unreachable")
	}

	var got []error
	deadline := time.After(time.Second)
	for len(got) < n-1 {
		select {
		case err := <-errs:
			got = append(got, err)
		case <-deadline:
			t.Fatalf("dropped records: got %d, want %d", len(got), n-1)
		}
	}
	for _, err := range got {
		if !errors.Is(err, kgo.ErrMaxBuffered) {
			t.Errorf("err = %v, want ErrMaxBuffered", err)
		}
	}
}
