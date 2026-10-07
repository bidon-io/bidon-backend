# 1 — Telemetry events logger

## Linear issue

**Title:** `POC: typed telemetry-events logger`

**User story.** As a backend engineer, I need a logger that writes **typed** catalog events (shared envelope + event-specific fields) to `telemetry-events`, instead of the unstructured JSON `AdEvent` blobs on `ad-events`.

**Goal.** `bidon-sdkapi` can fire-and-forget a typed catalog event onto `telemetry-events`. `ad-events` stays as it is.

> **Shipped with issue 2.** This issue was folded into the BAC-61 MR: the catalog went straight to protobuf with Confluent Schema Registry framing instead of the JSON `Record` first drafted here, because the warehouse (TRD) is protobuf on the bus anyway and the JSON shape would have been thrown away one MR later. The sections below describe what landed.

**Definition of done.**

- `internal/telemetry` logger: envelope + typed message per event, enqueue, do not await Kafka.
- Topic `telemetry-events` wired (`KAFKA_TELEMETRY_EVENTS_TOPIC`, default `telemetry-events`).
- Every catalog message embeds the `Envelope` (`event_id`, `event_name`, `event_ts`, `schema_version`, `app_id`, `auction_id`, …).
- Values are `proto.Marshal` of the catalog message; when `SCHEMA_REGISTRY_URL` is set they are Confluent-framed. Every record carries headers `event_name` and `protobuf_message`.
- Missing topic env, registry down, or produce error: log, no panic, caller not blocked.
- `ad-events` producer unchanged.

**Out of scope.** Ingest HTTP API, sampling, Connect / Parquet sink, framing in prod compose.

---

## MR instructions

### Do not

- Extend `internal/sdkapi/event.AdEvent` or `NotificationEvent`.
- Add a feature flag or `Enabled` switch — if Kafka is on, this logger is on.
- Put a `map[string]any` or opaque `payload` field on any catalog message.
- Block `/v2/auction` on Kafka or the Schema Registry.

### Schema

`schemas/proto/org/bidon/telemetry/v1/events.proto` (package `org.bidon.telemetry.v1`), generated into `pkg/proto/org/bidon/telemetry/v1`. Embedded as `schemas/proto.EventsProto` so the logger can register it.

- `Envelope` — identity header embedded as field 1 of every event.
- One message per event: `AuctionRequestReceived`, `AuctionCompleted`, `DspRequestSent`, `DspResponseReceived`, `DspResponseRejected`.
- Closed sets are enums: `Scope`, `Outcome`, `RejectReason`, `ErrorCode` (each with a `*_UNSPECIFIED = 0`).

`event_id` = UUID v4. `event_ts` = `time.Now().UnixMilli()`. `schema_version` = `"0.1"`. `sampling_rate` = `1.0`.

Lint with `buf lint schemas/proto` (STANDARD).

### Package layout

```
internal/telemetry/
  envelope.go         // EventName constants, attrs, Envelope (decode view)
  catalog.go          // Event.AuctionRequestReceived / AuctionCompleted / DSPRequestSent / DSPResponseReceived
  outcome.go          // Outcome / Scope / ErrorCode / RejectReason / AuctionResult Go types
  event.go            // proto <-> Go mapping, Record (decode-only view for tests and the log engine)
  logger.go           // Logger, Event.emit, LoggerEngine / LogMessage
  registry.go         // event name -> message type, headers, DecodeRecord
  confluent.go        // Confluent wire framing, schema id cache, fail-open
  schema_registry.go  // franz-go sr client, BACKWARD compatibility
  kafka.go log.go memory.go  // engines: Kafka, no-Kafka zap log, in-memory for tests
  metrics.go          // Prometheus counters (issue 2)
```

### Logger

Callers use typed methods on `Logger.Event`, which build the generated `telemetryv1` structs directly — no intermediate mapper layer:

```go
tel.Event.AuctionRequestReceived(params)
tel.Event.DSPResponseReceived(params, demandResponse)
```

`Event.emit` does `proto.Marshal` → optional Confluent frame → `Engine.Produce` with headers. Never wait on Kafka. Produce errors go to zap with the envelope fields.

When `USE_KAFKA` is false, use the `Log` engine (zap, decodes the record for readable output). `telemetry.Nop` discards.

### Schema Registry

See `schemas/proto/org/bidon/telemetry/v1/CONFLUENT.md`. Summary: subject strategy **TopicRecordNameStrategy** (`telemetry-events-org.bidon.telemetry.v1.<Message>`), compatibility **BACKWARD**, ids cached per subject, registry failures fail open to raw protobuf. Dev compose enables Redpanda’s registry on `:8081` (host `:18081`) and Console decodes from it.

### Kafka config

`config/kafka.go`:

```go
const TelemetryEventsTopic Topic = "telemetry_events"
```

```go
TelemetryEventsTopic: envOr("KAFKA_TELEMETRY_EVENTS_TOPIC", "telemetry-events"),
conf.SchemaRegistryURL = strings.TrimSpace(os.Getenv("SCHEMA_REGISTRY_URL"))
```

`.env.sample` + compose dev / staging:

```
KAFKA_TELEMETRY_EVENTS_TOPIC: telemetry-events
SCHEMA_REGISTRY_URL: http://redpanda:8081
```

### Wire-up (`cmd/bidon-sdkapi/main.go`)

Construct `telemetry.Logger` next to `event.Logger` on the same `kgo.Client` — do not open a second broker connection. Call `Logger.UseSchemaRegistry(url, topic)` when the URL is set. Pass it into `auction.Service` as `Telemetry`.

### Tests

- `logger_test.go`: memory engine records one value per emit on `TelemetryEventsTopic`; headers present; `DecodeRecord` round-trips framed and raw values; envelope fields populated.
- `confluent_test.go`: framing, message index, id caching, fail-open on registry error.
- Nil engine / `Nop`: no panic.

### Files to touch

- `schemas/proto/` (new), `pkg/proto/org/bidon/telemetry/v1/` (generated), `buf.yaml`
- `internal/telemetry/` (new)
- `config/kafka.go`
- `cmd/bidon-sdkapi/main.go`
- `.env.sample`
- `docker-compose.dev.yml`, staging/prod compose: topic + registry env
