# Confluent Schema Registry

Catalog produce is **raw protobuf** plus Kafka headers `event_name` and
`protobuf_message` when `SCHEMA_REGISTRY_URL` is empty.

When the URL is set, `Event.emit` registers `events.proto` and prefixes
each value with the **Confluent protobuf wire format** so Redpanda Console
and Connect `ProtobufConverter` can decode it:

`0x00` + 4-byte big-endian schema id + message indexes + `proto.Marshal`
payload.

Subject strategy is **TopicRecordNameStrategy**: `{topic}-{FullName}`
(e.g. `telemetry-events-org.bidon.telemetry.v1.DspResponseReceived`).
Compatibility is **BACKWARD**.

Registry failures **fail open**: log and produce raw proto. Do not block
`/v2/auction`. Schema ids are cached after the first successful register
of each type.

## Where

| Site | Job |
| --- | --- |
| [`internal/telemetry/logger.go`](../../../../../internal/telemetry/logger.go) `Event.emit` | After `proto.Marshal`, `confluentSerde.frame` when a registry is attached. |
| [`internal/telemetry/confluent.go`](../../../../../internal/telemetry/confluent.go) | Framing, message-index path, id cache, fail-open. |
| [`internal/telemetry/schema_registry.go`](../../../../../internal/telemetry/schema_registry.go) | `sr.Client` register + BACKWARD compatibility. |
| [`internal/telemetry/registry.go`](../../../../../internal/telemetry/registry.go) `DecodeRecord` | `stripConfluentPrefix` so tests and the log engine read framed or raw values. |
| [`config/kafka.go`](../../../../../config/kafka.go) `Kafka()` | Reads `SCHEMA_REGISTRY_URL`. |
| [`cmd/bidon-sdkapi/main.go`](../../../../../cmd/bidon-sdkapi/main.go) | `Logger.UseSchemaRegistry` next to `telemetry.Kafka` (same `kgo.Client`). |
| [`docker-compose.dev.yml`](../../../../../docker-compose.dev.yml) | Redpanda Schema Registry on `:8081` / host `:18081`. Console `KAFKA_SCHEMAREGISTRY_*`. sdkapi `SCHEMA_REGISTRY_URL=http://redpanda:8081`. |
| [`docker-compose.staging.yml`](../../../../../docker-compose.staging.yml) | Same registry on Redpanda `:8081` (in-compose only). sdkapi + Console use `${SERVICE_NAME_REDPANDA:-redpanda}`. |

## Out of this change

Connect S3 Parquet sink, RisingWave protobuf source, sampling, turning
framing on in staging/prod compose.
