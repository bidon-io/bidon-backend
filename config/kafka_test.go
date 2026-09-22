package config

import "testing"

func TestKafkaTelemetryTopicDefault(t *testing.T) {
	t.Setenv("KAFKA_TELEMETRY_EVENTS_TOPIC", "")

	conf, err := Kafka()
	if err != nil {
		t.Fatal(err)
	}
	if got := conf.Topics[TelemetryEventsTopic]; got != "telemetry-events" {
		t.Fatalf("default topic: got %q, want telemetry-events", got)
	}
}

func TestKafkaTelemetryTopicOverride(t *testing.T) {
	t.Setenv("KAFKA_TELEMETRY_EVENTS_TOPIC", "custom-telemetry")

	conf, err := Kafka()
	if err != nil {
		t.Fatal(err)
	}
	if got := conf.Topics[TelemetryEventsTopic]; got != "custom-telemetry" {
		t.Fatalf("override topic: got %q, want custom-telemetry", got)
	}
}
