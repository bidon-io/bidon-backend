package config

import (
	"context"
	"fmt"
	"os"
	"strings"

	sentryotel "github.com/getsentry/sentry-go/otel"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func ConfigureOTel() {
	tp := sdktrace.NewTracerProvider(
		tracerProviderOptions(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"), newOTLPTraceExporter)...,
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(sentryotel.NewSentryPropagator())
}

type otlpTraceExporterFactory func(ctx context.Context, endpoint string) (sdktrace.SpanExporter, error)

func newOTLPTraceExporter(ctx context.Context, endpoint string) (sdktrace.SpanExporter, error) {
	return otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(endpoint))
}

func tracerProviderOptions(endpoint string, newExporter otlpTraceExporterFactory) []sdktrace.TracerProviderOption {
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithSpanProcessor(sentryotel.NewSentrySpanProcessor()),
	}

	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return opts
	}

	exp, err := newExporter(context.Background(), endpoint)
	if err != nil {
		otel.Handle(fmt.Errorf("otlp trace exporter: %w", err))
		return opts
	}

	return append(opts, sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(exp)))
}
