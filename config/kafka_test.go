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

func TestKafkaProduceAdEvents(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want bool
	}{
		{name: "unset", env: "", want: false},
		{name: "false", env: "false", want: false},
		{name: "true", env: "true", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("KAFKA_PRODUCE_AD_EVENTS", tt.env)
			conf, err := Kafka()
			if err != nil {
				t.Fatal(err)
			}
			if conf.ProduceAdEvents != tt.want {
				t.Fatalf("ProduceAdEvents = %v, want %v", conf.ProduceAdEvents, tt.want)
			}
		})
	}
}
