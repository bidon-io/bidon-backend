package config

import (
	"context"
	"errors"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTracerProviderOptions_EmptyEndpointSkipsOTLP(t *testing.T) {
	called := false
	opts := tracerProviderOptions("", func(context.Context, string) (sdktrace.SpanExporter, error) {
		called = true
		return nil, nil
	})
	if called {
		t.Fatal("OTLP exporter factory must not be called when OTEL_EXPORTER_OTLP_ENDPOINT is unset")
	}
	if len(opts) != 1 {
		t.Fatalf("got %d tracer provider options, want 1 (Sentry only)", len(opts))
	}
}

func TestTracerProviderOptions_BlankEndpointSkipsOTLP(t *testing.T) {
	called := false
	opts := tracerProviderOptions("  \t", func(context.Context, string) (sdktrace.SpanExporter, error) {
		called = true
		return nil, nil
	})
	if called {
		t.Fatal("OTLP exporter factory must not be called when OTEL_EXPORTER_OTLP_ENDPOINT is blank")
	}
	if len(opts) != 1 {
		t.Fatalf("got %d tracer provider options, want 1 (Sentry only)", len(opts))
	}
}

func TestTracerProviderOptions_WithEndpointAddsOTLP(t *testing.T) {
	var gotEndpoint string
	opts := tracerProviderOptions("http://otelcol:4318", func(_ context.Context, endpoint string) (sdktrace.SpanExporter, error) {
		gotEndpoint = endpoint
		return tracetest.NewInMemoryExporter(), nil
	})
	if gotEndpoint != "http://otelcol:4318" {
		t.Fatalf("exporter endpoint = %q, want http://otelcol:4318", gotEndpoint)
	}
	if len(opts) != 2 {
		t.Fatalf("got %d tracer provider options, want 2 (Sentry + OTLP)", len(opts))
	}
}

func TestTracerProviderOptions_ExporterErrorKeepsSentryOnly(t *testing.T) {
	opts := tracerProviderOptions("http://otelcol:4318", func(context.Context, string) (sdktrace.SpanExporter, error) {
		return nil, errors.New("dial otelcol")
	})
	if len(opts) != 1 {
		t.Fatalf("got %d tracer provider options, want 1 (Sentry only after exporter error)", len(opts))
	}
}
