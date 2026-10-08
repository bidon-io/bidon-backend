package telemetry

import (
	"errors"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/bidon-io/bidon-backend/config"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event/engine"
)

// unreachableKafka returns a producer whose records never leave the buffer,
// as when the brokers are down.
func unreachableKafka(t *testing.T, opts ...kgo.Opt) *engine.Kafka {
	t.Helper()
	client, err := kgo.NewClient(append([]kgo.Opt{kgo.SeedBrokers("127.0.0.1:1")}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Close)
	return &engine.Kafka{
		Client: client,
		Topics: map[config.Topic]string{config.TelemetryEventsTopic: "telemetry-events"},
	}
}

// produceN produces n records and returns the errors reported for them,
// failing if Produce blocks.
func produceN(t *testing.T, k *Kafka, n int) []error {
	t.Helper()
	errs := make(chan error, n)
	returned := make(chan struct{})
	go func() {
		for i := 0; i < n; i++ {
			k.Produce(LogMessage{Topic: config.TelemetryEventsTopic, Value: []byte{0x0a}}, func(err error) {
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
	for {
		select {
		case err := <-errs:
			got = append(got, err)
		case <-time.After(200 * time.Millisecond):
			return got
		}
	}
}

func TestKafkaDoesNotBlockWhenClientBufferFull(t *testing.T) {
	k := &Kafka{Producer: unreachableKafka(t, kgo.MaxBufferedRecords(1))}

	errs := produceN(t, k, 3)
	if len(errs) != 2 {
		t.Fatalf("dropped records: got %d, want 2", len(errs))
	}
	for _, err := range errs {
		if !errors.Is(err, ErrBufferFull) {
			t.Errorf("err = %v, want ErrBufferFull", err)
		}
	}
}
