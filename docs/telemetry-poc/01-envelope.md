# 1 — Envelope and flagged telemetry logger

## Linear issue

**Title:** `POC: telemetry envelope and flagged telemetry-events logger`

**User story.** As a backend engineer, I want a separate, privacy-safe place to emit auction telemetry so we can add events later without widening `AdEvent` or touching today’s `ad-events` consumers.

**Goal.** `bidon-sdkapi` can fire-and-forget one JSON record onto a new Redpanda topic when a flag is on, and does nothing when the flag is off. No auction behaviour changes in this issue.

**Definition of done.**

- New package exists; logger does not block the caller.
- Flag default off; on only in local compose.
- Flag off + Kafka on → zero writes to the new topic.
- Flag on + missing topic env → log, no panic, no delayed handler.
- Marshalled payload includes the v0 envelope and none of: `idfa`, `idfv`, `ip`, `city`, `user_agent`, `raw_request`, `raw_response`.
- `ad-events` producer unchanged.

**Out of scope.** Call sites in auction/bidding, protobuf, ingest HTTP API.

---

## MR instructions

Implement only the produce plumbing. Do not instrument `auction.Service` or `bidding.Builder`.

### Do not

- Extend `internal/sdkapi/event.AdEvent` or `NotificationEvent`.
- Add Schema Registry / protobuf.
- `depends_on` any new observe service.

### Package layout

```
internal/telemetry/
  envelope.go      // Envelope + EventName constants
  event.go         // Record (envelope + optional type fields), Topic(), json tags
  logger.go        // Logger / LoggerEngine / LogMessage — copy the shape of internal/sdkapi/event
  logger_test.go
  privacy_test.go  // marshal must not contain PII keys
```

Reuse the existing Kafka engine if you can do it without importing `event` into a cycle. Prefer:

- `internal/telemetry.Logger` with the same `Log(ev Event, handleErr func(error))` contract.
- In `cmd/bidon-sdkapi/main.go`, when `USE_KAFKA=true`, construct **one** `kgo.Client` and either:
  - give `engine.Kafka` a topics map that includes the new topic and share it, or
  - duplicate the small produce wrapper under `internal/telemetry/engine` that takes `map[config.Topic]string`.

Sharing `internal/sdkapi/event/engine.Kafka` is fine if you introduce a shared `Produce` interface in `config` or pass `kgo.Client` in. Do not fork a second broker connection unless you have to.

### Envelope (v0)

```go
type Envelope struct {
    EventID        string  `json:"event_id"`
    EventName      string  `json:"event_name"`
    EventTS        int64   `json:"event_ts"`         // unix ms
    SchemaVersion  string  `json:"schema_version"`   // "0.1"
    AppID          int64   `json:"app_id"`
    AuctionID      string  `json:"auction_id"`
    SessionID      string  `json:"session_id"`
    AdType         string  `json:"ad_type,omitempty"`
    AdFormat       string  `json:"ad_format,omitempty"`
    Country        string  `json:"country,omitempty"` // MaxMind country code only
    TraceID        string  `json:"trace_id,omitempty"`
    SamplingRate   float64 `json:"sampling_rate"`    // always 1.0
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

### Record shape (wide JSON)

One struct for the POC so RisingWave can `ENCODE JSON` later. Type-specific fields `omitempty`:

```go
type Record struct {
    Envelope
    Scope             string  `json:"scope,omitempty"` // e.g. "bidding_round"
    DSP               string  `json:"dsp,omitempty"`
    Outcome           string  `json:"outcome,omitempty"`
    HTTPStatus        int     `json:"http_status,omitempty"`
    LatencyMS         int64   `json:"latency_ms,omitempty"`
    Price             float64 `json:"price,omitempty"`
    PriceFloor        float64 `json:"price_floor,omitempty"`
    WinnerDSP         string  `json:"winner_dsp,omitempty"`
    ParticipantCount  int     `json:"participant_count,omitempty"`
    TotalLatencyMS    int64   `json:"total_latency_ms,omitempty"`
    ErrorCode         string  `json:"error_code,omitempty"`
    RejectReason      string  `json:"reject_reason,omitempty"`
}

func (r Record) Topic() config.Topic { return config.TelemetryEventsTopic }
```

Add a constructor `NewRecord(name string, env Envelope) Record` that sets `EventName`, `SchemaVersion`, `SamplingRate`.

### Kafka config

In `config/kafka.go`:

```go
const TelemetryEventsTopic Topic = "telemetry_events"
```

Add to `conf.Topics`:

```go
TelemetryEventsTopic: os.Getenv("KAFKA_TELEMETRY_EVENTS_TOPIC"),
```

`.env.sample` + compose.dev kafka env:

```
KAFKA_TELEMETRY_EVENTS_TOPIC: telemetry-events
TELEMETRY_ENABLED: "true"    # compose.dev only
```

Default `TELEMETRY_ENABLED` unset/false everywhere else.

### Logger

```go
type Event interface {
    Topic() config.Topic
}

type Logger struct {
    Enabled bool
    Engine  LoggerEngine
}

func (l *Logger) Log(ev Event, handleErr func(error)) {
    if l == nil || !l.Enabled || l.Engine == nil {
        return
    }
    // json.Marshal + Engine.Produce, same as event.Logger
}
```

Never wait on Kafka. Use `Produce` callback like `internal/sdkapi/event/engine/kafka.go`.

### Wire-up (`cmd/bidon-sdkapi/main.go`)

```go
telemetryEnabled := os.Getenv("TELEMETRY_ENABLED") == "true"
telemetryLogger := &telemetry.Logger{Enabled: telemetryEnabled, Engine: /* same or sibling engine */}
```

Hold the logger on a variable. Do **not** attach it to `auction.Service` / `bidding.Builder` yet (that is issue 2). You may leave it unused except health if that is awkward — unused is acceptable; `var _ = telemetryLogger` is not. Prefer a private field on nothing: just construct it so the next MR can pass it. If the compiler complains, pass it into `auction.Service` as an optional `Telemetry *telemetry.Logger` **without calling it**.

### Tests

- `privacy_test.go`: build a record as if from a full `schema.BaseRequest` (include IDFA/IP in the *input* helper), marshal, `require.NotContains` those JSON keys and values.
- `logger_test.go`: `Enabled: false` → mock engine `Produce` count 0. `Enabled: true` + engine that records messages → one JSON blob, topic key `TelemetryEventsTopic`.
- Empty topic string: engine should `handleErr`, not panic (today’s Kafka engine already errors if topic unset).

### Existing files to touch

- `config/kafka.go`
- `cmd/bidon-sdkapi/main.go`
- `.env.sample`
- `docker-compose.dev.yml` (`x-dev-kafka-env`)
- `docker-compose.staging.yml` / prod compose: add topic env, **leave flag off**
