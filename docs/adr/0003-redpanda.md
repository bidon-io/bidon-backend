# ADR-0003: Redpanda for dev and staging

**Status:** Accepted
**Date:** 2026-06-19
**Linear:** [BAC-13](https://linear.app/bidon/issue/BAC-13/replace-kafka-with-redpanda-in-dev-and-enable-geolite2-geocoding)

---

## Context

Dev and staging ran Confluent Kafka plus Zookeeper. That is a JVM and a second process for a bus the services only use as a log of auction and notification events. Local startup was the painful part: Zookeeper had to be healthy before Kafka, and the advertised listeners were easy to get wrong.

The producers and consumers already speak the Kafka protocol. Telemetry later needs a schema registry beside the broker so protobuf values can be Confluent-framed.

## Decision

Dev (`docker-compose.dev.yml`) and staging (`docker-compose.staging.yml`) use one Redpanda process as the Kafka API broker, with Schema Registry on the same process and Redpanda Console for inspection.

Clients keep `KAFKA_BROKERS_LIST`. Nothing in the Go services or the Rust `bidon-telemetry` ingest is Redpanda-specific.

Rejected: keep Kafka and Zookeeper and only document them better. Also rejected: a non-Kafka bus, which would rewrite every producer.

## Consequences

- `docker-compose-prod.yml` and the older `docker-compose.yml` still run Confluent Kafka and Zookeeper. Prod was out of scope for BAC-13. Do not "fix" those files into Redpanda without a separate decision.
- On Coolify, Redpanda's advertised hostname is the preview container name. Schema Registry dials that name, and the container's own bridge IP never completes, so the staging entrypoint rewrites that hosts line to `127.0.0.1`. Other containers still use Docker DNS.
- Console on staging is public through Traefik and is behind basic auth (`REDPANDA_CONSOLE_BA_USERS`).
