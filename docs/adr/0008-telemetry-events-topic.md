# ADR-0008: Server telemetry on its own topic

**Status:** Accepted
**Date:** 2026-10-27
**Linear:** [BAC-60](https://linear.app/bidon/issue/BAC-60/poc-typed-telemetry-events-logger), [BAC-61](https://linear.app/bidon/issue/BAC-61/poc-auction-and-dsp-events-plus-metrics-counters)

---

## Context

Auction and DSP outcomes were inferred from the JSON `ad-events` topic, or not recorded at all. That topic is a grab bag: its shape is not a catalog, and a produce failure was allowed to sit on the auction path.

The warehouse and Redpanda Console need a schema per message. A JSON topic has nothing to register.

## Decision

Server bidding-round events go to `telemetry-events` as Confluent-framed protobuf. The catalog is `schemas/proto/org/bidon/telemetry/v1/events.proto`. Each message carries an `Envelope` and declares its wire name with `option (event_name)`. Emit methods on `Telemetry.Event()` are the only writers. A nil `Telemetry` is a no-op.

The five messages are `AuctionRequestReceived`, `DspRequestSent`, `DspResponseReceived`, `DspResponseRejected`, and `AuctionCompleted`. One request, one sent and one received per DSP, one completion. A bid at or below the floor is `dsp_response_rejected`.

When `SCHEMA_REGISTRY_URL` is set, values use the Confluent header and `TopicRecordNameStrategy` subjects (`telemetry-events-` plus the protobuf full name), registered as BACKWARD. If the registry is down, the producer writes raw protobuf and the auction still returns. Produce never waits on Kafka or the registry.

`ad-events` stays a separate JSON topic. `KAFKA_PRODUCE_AD_EVENTS` defaults to off. Notification events, ad events, and telemetry share one Kafka client. Without Kafka, the dev engine logs each catalog event as protojson.

Prometheus counters on sdkapi `/metrics` are low cardinality: `dsp_response_total{dsp,outcome}`, `dsp_request_duration_seconds{dsp}`, `auction_completed_total{result}`. No `auction_id` label.

Rejected: extending `ad-events` with more JSON. Also rejected: failing the auction when Kafka or the registry is down.

## Consequences

- Two topics describe overlapping moments. `ad-events` is off unless someone turns it on. New server facts belong on `telemetry-events`, not on a new JSON event.
- A consumer that requires the Confluent header must tolerate raw protobuf from the period before the registry answered. Once a subject is registered the value is that header (magic byte, schema id, zigzag message indexes) plus the protobuf payload. Redpanda only hosts the registry in dev and staging. The header is the Confluent Kafka wire format.
- Sampling is recorded as `1.0`. A later sampler has to keep every event for one auction together, and must not sample funnel stages, errors, impressions, or notices.
- Client ingest (`POST /v2/telemetry`) is the separate `bidon-telemetry` process and writes to the same topic. See [ADR-0002](0002-telemetry-ingest.md). This ADR is only the server catalog.
