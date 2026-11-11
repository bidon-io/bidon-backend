# ADR-0002: SDK telemetry ingest service

**Status:** Accepted
**Date:** 2026-11-04
**Deciders:** Zandre Witte
**Linear:** [BAC-73](https://linear.app/bidon/issue/BAC-73), [BAC-76](https://linear.app/bidon/issue/BAC-76), [BAC-91](https://linear.app/bidon/issue/BAC-91)
**Supersedes / Superseded by:** —
**Implementation:** [PR #963](https://github.com/bidon-io/bidon-backend/pull/963) (`bac-73-sdk-telemetry-ingest`). Status of what that PR does and does not cover: [docs/telemetry/README.md](../telemetry/README.md).

---

## Context

Client telemetry is a different job from running an auction. The SDK posts a
batch of already-named events and a few metric samples. The server checks the
names, stamps the envelope, and produces protobuf to the same `telemetry-events`
topic the auction logger already writes. Billing, show, win, and loss stay on
their existing routes.

[BAC-76](https://linear.app/bidon/issue/BAC-76) first described this as a thin
handler inside `bidon-sdkapi`, next to `/v2/show`. Its done line is the
constraint that matters: `/v2/auction` is unchanged if ingest fails. The batch
is untrusted JSON, capped but still large next to a bid request (256 KiB, 100
events, 50 metrics). Produce is fire-and-forget. An unknown name is dropped
and the batch still returns 202. Schema Registry and Kafka must not sit on the
response.

The auction logger already lives in-process in `internal/telemetry`. It frames
protobuf with the Confluent header and registers schemas off the request path
(`internal/telemetry/confluent.go`). Client ingest has to emit records that
consumer can read beside those auction rows. It does not need Postgres, Redis,
the bidding stack, or the app's `BaseRequest`.

---

## Decision

Serve `POST /v2/telemetry` from **`bidon-telemetry`**, a Rust executable in
`telemetry/`, on port **1326**. `bidon-sdkapi` does not register the route.
The OpenAPI contract lives in `telemetry/openapi.yaml`, separate from the sdkapi
spec, so sdkapi neither documents nor generates code for it. `internal/sdkapi/v2/app/app_test.go`
fails if sdkapi grows the route.

Auction and DSP events stay in the Go logger. Both writers use topic
`telemetry-events`, the `event_name` and `protobuf_message` headers, and the
same Confluent framing (magic `0x00`, schema id, zigzag message indexes,
payload). `telemetry/src/frame.rs` is the Rust copy of that framing.

### Decisions

| #  | Decision | Rejected alternative |
| -- | -------- | -------------------- |
| D1 | A separate process. Client ingest cannot share the auction process's memory, request deadlines, or Kafka producer. Stopping, crashing, or blocking `bidon-telemetry` leaves `/v2/auction` up. The kill switch is `TELEMETRY_INGEST_ENABLED` on this process only. | The thin sdkapi handler BAC-76 first described. A handler would meet the HTTP contract, and a panic, a full produce buffer, or a stuck Kafka client would sit in the auction process. |
| D2 | The executable is Rust (`telemetry/`, Axum, prost, prost-reflect, rskafka). It loads `events.proto` as a descriptor and walks protojson. A new client message is a proto `event_name` option; there is no per-event handler. The crate does not import the Bidon Go module. | A second Go binary under `cmd/` that links `internal/telemetry` and `go.mod`. That would also isolate the process. It was not taken: this proxy is a protojson-to-framed-protobuf encoder, and the auction logger already owns the Go side of the topic. |
| D3 | One image and one port. `telemetry/Dockerfile` builds a static-ish bookworm binary. Compose runs `rust:1.98-bookworm` with `telemetry/dev-entrypoint.sh` (`cargo run`) on `1326:1326`. Staging uses `BIDON_TELEMETRY_TAG`; there is no prod deployment yet. Health is `GET /health`. Metrics are `GET /metrics` on the same port. VictoriaMetrics scrapes sdkapi `:1324` and this process `:1326`. | Serving `/metrics` from sdkapi. Client observations would then depend on the auction process being the thing that parsed the batch. |
| D4 | The HTTP handler returns without waiting on Kafka or Schema Registry. Events go into a bounded channel (4096). A full channel counts `buffer_full` and the batch is still 202. A background task produces in batches of up to 500 records and 512 KiB with lz4. Kafka retries stop after 5 seconds; a failed produce counts `produce_error` for each record in the batch and is logged. On SIGTERM or Ctrl-C the server stops accepting, open requests finish, the queue is drained, and the process exits within 8 seconds. Schema registration retries in the background (1s growing to 30s, 2s HTTP timeout). Until a subject has an id, the value is raw protobuf. | Awaiting the produce, or failing the request when the registry is down. That puts Redpanda on the SDK's latency and turns a telemetry outage into a client error. |
| D5 | Identity is the JSON `app.key`, `app.bundle`, and `session.id` (UUID). There is no database lookup. `app_id` is stamped `0`, `country` and `trace_id` are cleared, and `schema_version` is int32 `1`. `app_key` and `bundle` are written on `envelope.app`. Flat client labels `host` and `auction_owner` are stored on `envelope.integration` as the PRD enums. | Resolving the app row, filling `country` from MaxMind, and deduping `event_id` in this service. The spike lists those (BE-M0-3). They need Postgres, the geocoder, and Redis, which would pin ingest to the same dependencies as sdkapi. They stay follow-ups. |
| D6 | Server-minted names are refused: `auction_request_received`, `auction_completed`, `dsp_request_sent`, `dsp_response_received`, `dsp_response_rejected`. Unknown event names, unknown metric names, and extra metric labels are dropped. Only the three client histograms are applied, with label values from fixed sets and non-negative finite values. An event that matches the catalogue but fails to decode counts `dropped_invalid`. 413 is the only size reject. Malformed JSON and a missing app or session are 400. | Accepting whatever `event_name` the client sends, or 400-ing the whole batch for one unknown name. |

### What the process does on one request

1. Read the body up to 256 KiB. Larger is 413. More than 100 events or 50 metrics is 413.
2. Require `app.key`, `app.bundle`, and a UUID `session.id`.
3. If ingest is disabled, return `{"accepted":0}` and produce nothing.
4. For each event, encode it only when `envelope.event_name` is in the client catalogue. Stamp the envelope. Frame it. `try_enqueue` it.
5. For each metric, observe it only when the name, type (`histogram`), and labels match an allow-listed series.
6. Return 202 with the number of items enqueued or observed.

### Components

| Piece | Role |
| ----- | ---- |
| `telemetry/src/main.rs` | Load env, bind `:1326`, start the producer task and the schema-registration task |
| `telemetry/src/http.rs` | `POST /v2/telemetry`, `GET /metrics`, `GET /health` |
| `telemetry/src/ingest.rs` | Batch limits, allow-list, enqueue, kill switch |
| `telemetry/src/catalog.rs` | prost-reflect catalogue from `events.proto`; stamp `envelope.app`, `schema_version`, integration enums |
| `telemetry/src/frame.rs` | Confluent protobuf framing, matching `internal/telemetry/confluent.go` |
| `telemetry/src/registry.rs` | Background schema registration; raw protobuf until an id exists |
| `telemetry/src/kafka.rs` | rskafka producer on partition 0 (the topic must already exist); headers `event_name` and `protobuf_message` |
| `telemetry/src/metrics.rs` | `ingest_total{result}` and the three client histograms |

### Configuration

| Env | Default | Meaning |
| --- | ------- | ------- |
| `TELEMETRY_PORT` | `1326` | Listen port |
| `KAFKA_BROKERS_LIST` | required | Redpanda brokers |
| `KAFKA_TELEMETRY_EVENTS_TOPIC` | `telemetry-events` | Same topic as the Go logger |
| `KAFKA_CLIENT_ID` | `bidon-telemetry` | Kafka client id |
| `SCHEMA_REGISTRY_URL` | empty | Empty skips registration; values stay raw protobuf |
| `TELEMETRY_INGEST_ENABLED` | `true` | `0`, `false`, `FALSE`, or `no` accepts nothing and still returns 202 |

Callers use a host that reaches this process, not sdkapi. `http-client.env.json`
sets `telemetryHost` to `http://localhost:1326` locally. sdkapi remains on
`1324`.

---

## Consequences

### Positive

- The auction path's failure domain stops at the Go process. Ingest load, a
  bad batch, a wedged produce, or a down registry shows up as `ingest_total`
  and logs on `:1326`.
- The catalogue is the proto. Adding a client event does not add a Go handler
  or an sdkapi route.
- Auction rows and client rows share a topic and a framing, so one consumer
  reads both.
- The telemetry contract has its own OpenAPI file, so the sdkapi spec and its
  generated code are unchanged.
- The image does not link Postgres, Redis, or the bidding code. Compose only
  waits for Redpanda.

### Negative / accepted

- The repo now builds a second language. Local compose uses the Rust image and
  a cargo cache. Release builds go through `telemetry/Dockerfile` and
  `BIDON_TELEMETRY_TAG`, not the Go `just` targets that build sdkapi.
- Two Kafka clients write one topic (franz-go in Go, rskafka here). Framing
  has to stay in lockstep; `telemetry/src/frame.rs` and
  `internal/telemetry/confluent_test.go` are the checks.
- `app_id` is 0. Joins use `app_key` and `bundle`. Country is empty. There is
  no `event_id` dedupe. That is the POC, not the spike's full BE-M0-3.
- The topic `telemetry-events` is declared once by the `redpanda-topics` compose
  service (one partition, replication 1). Neither process creates it. The Rust
  producer writes only partition 0, so adding partitions needs a partitioner here
  first.
- The process connects to Kafka before it listens. If Kafka is down at start it
  exits and the orchestrator restarts it.
- Schema Registry downtime means raw protobuf on the topic until registration
  catches up. Consumers that require the Confluent header must tolerate that
  window.
- Ingest is unauthenticated. Anyone can post well-formed batches for any
  `app_key`, including apps that do not exist. The damage is junk events and
  skewed per-app numbers; the label allow-lists stop new metric series. The
  planned fix is a token issued by `/v2/config`
  ([BAC-125](https://linear.app/bidon/issue/BAC-125)).
- Clients and any edge proxy must send `POST /v2/telemetry` to this service.
  The path is not on `:1324`.
- `/v2/config` does not yet advertise `telemetry.enabled`, `sample_rate`, or
  the allow-lists. The kill switch is an env var on this process. Sampling
  stays on the SDK.

### Follow-ups

1. BAC-126: give `bidon-telemetry` its own staging hostname; sdkapi serves it as
   `telemetry.url`.
2. BAC-78: publish the config block from sdkapi, including `url`, which is how
   the SDK finds this service. Leave sampling on the SDK.
   This service keeps the env kill switch.
3. Optional `client_event_total{event_name}` on this process (BAC-77). Do not
   move the existing series onto sdkapi.
4. BAC-125: ingest auth with a `/v2/config` token. It
   also fills `app_id`. MaxMind `country` and `event_id` dedupe only if ingest
   is allowed to take those dependencies. They are not required for the POC.
5. Auction spans (BAC-62) and impression or notice outcomes (#961) stay in
   sdkapi. Do not fold them into this executable.
