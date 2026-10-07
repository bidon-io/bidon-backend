package engine

import (
	"testing"

	"github.com/twmb/franz-go/pkg/kgo"
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
