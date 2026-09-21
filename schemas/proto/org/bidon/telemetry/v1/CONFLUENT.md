# Confluent Schema Registry follow-up

This module is the **schema registry** for catalog events (`events.proto` →
`pkg/proto/org/bidon/telemetry/v1`). Produce today is **raw protobuf** plus
Kafka headers `event_name` and `protobuf_message`.

The warehouse TRD wants **Confluent wire format** on `telemetry-events`
(magic byte `0x00` + 4-byte big-endian schema id + protobuf payload) so
Connect `ProtobufConverter` can write Parquet. Do not implement that in the
ad path until a registry URL is configured; unset URL must keep this raw
produce so auctions stay unchanged.

## Where

| Site | Job |
| --- | --- |
| [`internal/telemetry/logger.go`](../../../../../internal/telemetry/logger.go) `Logger.Log` | After `proto.Marshal`, optionally wrap bytes with Confluent framing. Keep `NewMessage` / type headers for logs and tests. |
| [`internal/telemetry/kafka.go`](../../../../../internal/telemetry/kafka.go) `Kafka.Produce` | Or a sibling serde used here: look up / cache schema id, write framed `kgo.Record.Value`. Do not open a second broker client. |
| [`internal/telemetry/registry.go`](../../../../../internal/telemetry/registry.go) | Register each catalog type (`CatalogEventNames` → proto full name) against Schema Registry. Subject strategy **TopicRecordNameStrategy**: `{topic}-{FullName}` e.g. `telemetry-events-org.bidon.telemetry.v1.DspResponseReceived`. Compatibility **BACKWARD**. |
| [`internal/telemetry/memory.go`](../../../../../internal/telemetry/memory.go) / `DecodeRecord` | Strip the 5-byte prefix when present so tests keep decoding proto messages. |
| [`config/kafka.go`](../../../../../config/kafka.go) `Kafka()` | Read `SCHEMA_REGISTRY_URL` (empty = raw proto, current behaviour). |
| [`cmd/bidon-sdkapi/main.go`](../../../../../cmd/bidon-sdkapi/main.go) | Construct the serde next to `telemetry.Kafka` (same `kgo.Client`). Fail open if the registry is down: log and produce raw proto; do not block `/v2/auction`. |
| Compose | Schema Registry already exists in `docker-compose.yml` / `docker-compose-prod.yml`. Pass `SCHEMA_REGISTRY_URL` into **sdkapi** (not only the broker). `docker-compose.dev.yml` does not run a registry yet. |

## What

1. Client: `github.com/twmb/franz-go/pkg/sr` (already in the Franz stack) or an equivalent Confluent API client.
2. On first produce of each message type: register `events.proto` (or the generated descriptor) and cache the schema id. Later produces must not wait on HTTP.
3. Wire value: `0x00` + `schema_id` (4 bytes BE) + `proto.Marshal` payload. No JSON, no fat `oneof` wrapper (TRD §4.3).
4. Headers can stay; Connect keys off the schema id, not `event_name`.
5. Tests: unit-test framing round-trip with a fake registry. Do not require Schema Registry or otelcol in CI.

## Out of this follow-up

Connect S3 Parquet sink, RisingWave protobuf source, sampling, turning this
on by default in local compose.
