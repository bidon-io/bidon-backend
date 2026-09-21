package telemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTraceIDFromContext_EmptyWithoutSpan(t *testing.T) {
	if got := TraceIDFromContext(context.Background()); got != "" {
		t.Fatalf("TraceIDFromContext() = %q, want empty", got)
	}
}

func TestTraceIDFromContext_FromSpan(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(prev)
	})

	ctx, span := Tracer().Start(context.Background(), SpanAuctionRun)
	defer span.End()

	got := TraceIDFromContext(ctx)
	want := span.SpanContext().TraceID().String()
	if got != want {
		t.Fatalf("TraceIDFromContext() = %q, want %q", got, want)
	}

	params := Params{TraceID: "explicit"}
	if got := params.withContextTrace(ctx).TraceID; got != "explicit" {
		t.Fatalf("explicit TraceID overwritten: %q", got)
	}
	if got := (Params{}).withContextTrace(ctx).TraceID; got != want {
		t.Fatalf("withContextTrace() = %q, want %q", got, want)
	}
}
