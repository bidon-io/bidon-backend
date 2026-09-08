# 3 — Auction and DSP traces

## Linear issue

**Title:** `POC: OpenTelemetry auction parent and per-DSP child spans`

**User story.** As a backend engineer (or support looking at one partner complaint), I want the legs of a single auction as one trace so I can see which DSP was slow or timed out without reconstructing it from logs.

**Goal.** Each `/v2/auction` creates one parent span and one child per DSP. Events from that request carry `trace_id`. When an OTLP endpoint is configured, export there; keep Sentry as it is.

**Definition of done.**

- Test recorder: 3 DSPs → 1 parent + 3 children with `app_id`, `auction_id`, `dsp`, `outcome`.
- Telemetry events for that request share the parent `trace_id`.
- `OTEL_EXPORTER_OTLP_ENDPOINT` unset → no OTLP export; auctions unchanged.
- Manual (with issue 4): tree visible in VictoriaTraces.

**Out of scope.** Spanmetrics, sampling, turning Sentry off, product trace UI.

---

## MR instructions

Blocked on issue 1 (`trace_id` on the envelope). Pair with issue 4 for a destination. Can merge-request without compose if tests use `tracetest.SpanRecorder`.

### Today

`config/otel.go` installs **only** a Sentry span processor:

```go
tp := trace.NewTracerProvider(
    trace.WithSpanProcessor(sentryotel.NewSentrySpanProcessor()),
)
```

Do not remove Sentry in this MR.

### Tracer provider

When `os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")` is non-empty (e.g. `http://otelcol:4318`):

- Start an OTLP HTTP trace exporter (`go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp`) pointed at that endpoint (`WithEndpointURL` or host+path `/v1/traces`).
- `trace.NewTracerProvider` with **both** `sentryotel.NewSentrySpanProcessor()` and `trace.NewBatchSpanProcessor(otlpExporter)`.
- Keep `otel.SetTextMapPropagator` as today (Sentry propagator is fine for POC).

When unset: keep current Sentry-only provider.

Use a tracer name `bidon-sdkapi/auction`.

### Spans

**Parent** — `internal/auction/service.go` `Run`:

```go
ctx, span := tracer.Start(ctx, "auction.run",
    trace.WithAttributes(
        attribute.Int64("app_id", params.App.ID),
        attribute.String("auction_id", req.AdObject.AuctionID),
    ),
)
defer span.End()
```

Pass this `ctx` into `AuctionBuilder.Build` / bidding so children nest. Today `Build` already takes `ctx` — verify it reaches `HoldAuction` → `processAdapter`. If any hop drops context, fix that hop.

**Children** — `processAdapter`:

```go
ctx, span := tracer.Start(ctx, "auction.dsp",
    trace.WithAttributes(
        attribute.String("dsp", string(adapterKey)),
        attribute.String("auction_id", auctionRequest.AdObject.AuctionID),
    ),
)
defer span.End()
// after outcome known:
span.SetAttributes(attribute.String("outcome", outcome))
```

Child duration must cover `ExecuteRequest` / Amazon `FetchBids` + `ParseBids`.

### `trace_id` on events

When emitting telemetry (issue 2 call sites), set `Envelope.TraceID` from:

```go
oteltrace.SpanFromContext(ctx).SpanContext().TraceID().String()
```

If there is no span (tests without a provider), leave empty.

Issue 2 may land first without this; this MR adds the field fill. If you implement 2+3 together, do it once.

### Tests

`go.opentelemetry.io/otel/sdk/trace/tracetest`:

- Install a `TracerProvider` with `tracetest.NewSpanRecorder` in the auction/bidding test.
- Drive `Run` / `HoldAuction` with 3 stub adapters.
- Assert span names `auction.run` + `auction.dsp`, attribute keys, and parent-child links.

Do not require otelcol in CI.

### Caveat (accept)

Auction spans may also show up in Sentry. File a follow-up to unhook Sentry from this path if it is noisy; do not do it here.
