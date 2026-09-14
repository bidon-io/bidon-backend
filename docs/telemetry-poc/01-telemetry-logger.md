# 1 — Telemetry events logger

## Linear issue

**Title:** `POC: typed telemetry-events logger`

**User story.** As a backend engineer, I need a logger that writes **typed** catalog events (shared envelope + event-specific fields) to `telemetry-events`, instead of the unstructured JSON `AdEvent` blobs on `ad-events`.

**Goal.** `bidon-sdkapi` can fire-and-forget a typed record onto `telemetry-events`. No auction call sites in this issue — that is issue 2. `ad-events` stays as it is.

**Definition of done.**

- `internal/telemetry` logger: envelope + typed payload, enqueue, do not await Kafka.
- Topic `telemetry-events` wired (`KAFKA_TELEMETRY_EVENTS_TOPIC`).
- Marshalled JSON always has the envelope fields (`event_id`, `event_name`, `event_ts`, `schema_version`, `app_id`, `auction_id`, …).
- Missing topic env: log, no panic, caller not blocked.
- `ad-events` producer unchanged.

**Out of scope.** Auction/bidding emit sites, protobuf, ingest HTTP API.

---

## MR instructions

Produce plumbing only. Do not instrument `auction.Service` or `bidding.Builder`.

### Do not

- Extend `internal/sdkapi/event.AdEvent` or `NotificationEvent`.
- Add Schema Registry / protobuf.
- Add a feature flag or `Enabled` switch — if Kafka is on, this logger is on.

### Package layout

```
internal/telemetry/
  envelope.go      // Envelope + EventName constants
  event.go         // Record (envelope + typed fields), Topic(), json tags
  logger.go        // Logger / LoggerEngine / LogMessage — same shape as internal/sdkapi/event
  logger_test.go
```

Reuse the existing Kafka engine if you can without an import cycle:

- `internal/telemetry.Logger` with `Log(ev Event, handleErr func(error))`.
- One `kgo.Client` in `cmd/bidon-sdkapi/main.go`. Either extend `engine.Kafka`’s topics map or a thin sibling wrapper that takes `map[config.Topic]string`.
- Do not open a second broker connection.

### Envelope (v0)

```go
type Envelope struct {
    EventID       string  `json:"event_id"`
    EventName     string  `json:"event_name"`
    EventTS       int64   `json:"event_ts"`       // unix ms
    SchemaVersion string  `json:"schema_version"` // "0.1"
    AppID         int64   `json:"app_id"`
    AuctionID     string  `json:"auction_id"`
    SessionID     string  `json:"session_id"`
    AdType        string  `json:"ad_type,omitempty"`
    AdFormat      string  `json:"ad_format,omitempty"`
    Country       string  `json:"country,omitempty"`
    TraceID       string  `json:"trace_id,omitempty"`
    SamplingRate  float64 `json:"sampling_rate"`    // 1.0 for this POC
}

const SchemaVersion = "0.1"

const (
    EventAuctionRequestReceived = "auction_request_received"
    EventAuctionCompleted       = "auction_completed"
    EventDSPRequestSent         = "dsp_request_sent"
    EventDSPResponseReceived    = "dsp_response_received"
    EventDSPResponseRejected    = "dsp_response_rejected"
)
```

`event_id` = UUID v4. `event_ts` = `time.Now().UnixMilli()`.

### Record (typed, one JSON object)

Typed fields on the same struct so RisingWave can `ENCODE JSON`. Unused fields `omitempty` — they are still first-class columns, not a `map[string]any` or a JSON `payload` blob.

```go
type Record struct {
    Envelope
    Scope            string  `json:"scope,omitempty"` // e.g. "bidding_round"
    DSP              string  `json:"dsp,omitempty"`
    Outcome          string  `json:"outcome,omitempty"`
    HTTPStatus       int     `json:"http_status,omitempty"`
    LatencyMS        int64   `json:"latency_ms,omitempty"`
    Price            float64 `json:"price,omitempty"`
    PriceFloor       float64 `json:"price_floor,omitempty"`
    WinnerDSP        string  `json:"winner_dsp,omitempty"`
    ParticipantCount int     `json:"participant_count,omitempty"`
    TotalLatencyMS   int64   `json:"total_latency_ms,omitempty"`
    ErrorCode        string  `json:"error_code,omitempty"`
    RejectReason     string  `json:"reject_reason,omitempty"`
}

func (r Record) Topic() config.Topic { return config.TelemetryEventsTopic }
```

Constructor `NewRecord(name string, env Envelope) Record` sets `EventName`, `SchemaVersion`, `SamplingRate`.

### Kafka config

`config/kafka.go`:

```go
const TelemetryEventsTopic Topic = "telemetry_events"
```

```go
TelemetryEventsTopic: os.Getenv("KAFKA_TELEMETRY_EVENTS_TOPIC"),
```

`.env.sample` + compose.dev / staging kafka env:

```
KAFKA_TELEMETRY_EVENTS_TOPIC: telemetry-events
```

### Logger

```go
type Event interface {
    Topic() config.Topic
}

type Logger struct {
    Engine LoggerEngine
}

func (l *Logger) Log(ev Event, handleErr func(error)) {
    if l == nil || l.Engine == nil {
        return
    }
    // json.Marshal + Engine.Produce, same as event.Logger
}
```

Never wait on Kafka. `Produce` callback like `internal/sdkapi/event/engine/kafka.go`.

When `USE_KAFKA` is false, use a log/no-op engine (same split as today’s `event.Logger`).

### Wire-up (`cmd/bidon-sdkapi/main.go`)

Construct `telemetry.Logger` next to `event.Logger`. Do **not** call it from `auction.Service` / `bidding.Builder` yet (issue 2). Pass it into `auction.Service` as `Telemetry *telemetry.Logger` unused if that is the cleanest way to keep the binary compiling.

### Tests

- `logger_test.go`: mock engine records one JSON blob on `TelemetryEventsTopic`; envelope fields present; type fields are named columns.
- Empty topic string: `handleErr`, no panic.

### Files to touch

- `internal/telemetry/` (new)
- `config/kafka.go`
- `cmd/bidon-sdkapi/main.go`
- `.env.sample`
- `docker-compose.dev.yml`, staging/prod compose: topic env only
