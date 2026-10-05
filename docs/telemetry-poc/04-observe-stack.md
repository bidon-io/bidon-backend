# 4 — Local live funnel and observe stack

## Linear issue

**Title:** `POC: local RisingWave live funnel plus VM / VT / Grafana`

**User story.** As an engineer demoing telemetry (and later anyone watching auction health), I want to query auctions that just ran — in flight, DSP outcomes, a 5-minute funnel — in a browser or `psql`, without standing up S3 or DuckDB.

**Goal.** Dev compose runs standalone RisingWave as a consumer of `telemetry-events`, plus VictoriaMetrics, VictoriaTraces, otelcol, and Grafana. sdkapi does not depend on RisingWave. Stopping the observe services does not break ads.

**Definition of done.**

- `docker compose -f docker-compose.dev.yml up` starts the new services.
- After one auction (issue 2): row in Redpanda Console; `SELECT` from `funnel_5m` and `dsp_outcomes_5m`; `auctions_in_flight` populates then clears on `auction_completed`.
- Grafana shows VM rate panels and RisingWave funnel panels.
- With issue 3: paste `trace_id` into VictoriaTraces and see the tree.
- How-to doc: Console, `psql`, Grafana, VT.
- sdkapi returns 200 for `/v2/auction` if RisingWave / VM / VT are stopped.

**Out of scope.** Kafka Connect, Spaces, DuckDB, Alertmanager, Coolify, RisingWave cluster, Iceberg.

---

## MR instructions

Blocked on issue 1 (topic + catalog protobuf). Useful after issue 2. Do not add `depends_on: risingwave` to `bidon-sdkapi`.

### Compose (`docker-compose.dev.yml`)

Add services. Pin image tags (do not use `latest` without a digest). Suggested host ports: otelcol `4318`, VM `8428`, VT `10428`, RisingWave `4566`, Grafana `3001` (3010 is bidon-ui).

**RisingWave:** official standalone (`risingwavelabs/risingwave`, command `standalone` or the documented all-in-one). Local volume; wipe on reset is fine. Depends on `redpanda` healthy.

**risingwave-init:** oneshot `postgres`/`psql` client that waits for `4566` then applies `docker/telemetry/risingwave-init.sql`. Restart: `on-failure`.

**otelcol-contrib:** receive OTLP HTTP `:4318`, export traces to VictoriaTraces (OTLP). Optional Prometheus exporter unused if VM scrapes sdkapi directly.

**VictoriaMetrics:** scrape `bidon-sdkapi:1323/metrics` every 15s.

**VictoriaTraces:** default single-node; receive OTLP from the collector.

**Grafana:** provision datasources (Prometheus → `http://victoriametrics:8428`, Postgres → RisingWave `risingwave:4566` user `root` db `dev` — confirm defaults for the image you pin). Provision two dashboards (JSON under `docker/telemetry/grafana/`):

1. VM: `rate(dsp_response_total{outcome="timeout"}[5m])`, histogram p95 `dsp_request_duration_seconds`, `increase(auction_completed_total[5m])`.
2. RW: tables/stats for `funnel_5m`, `dsp_outcomes_5m`, `count(*)` from `auctions_in_flight`.

**sdkapi env:**

```
OTEL_EXPORTER_OTLP_ENDPOINT: http://otelcol:4318
```

(No-op until issue 3 lands.)

### RisingWave SQL (`docker/telemetry/risingwave-init.sql`)

`telemetry-events` carries one protobuf message type per event (`schemas/proto/org/bidon/telemetry/v1/events.proto`), not one wide JSON object. Each value is Confluent-framed (magic byte + schema id + message index) and registered under **TopicRecordNameStrategy**: `telemetry-events-org.bidon.telemetry.v1.<Message>`. Every record also carries Kafka headers `event_name` (e.g. `dsp_response_received`) and `protobuf_message` (the message full name). See `schemas/proto/org/bidon/telemetry/v1/CONFLUENT.md`.

So: **one source per message type** the views need, each pinned to its message via the registry. A source decodes every record on the topic with its one descriptor — a `DspRequestSent` read as `DspResponseReceived` decodes without error into wrong columns — so every source keeps the `protobuf_message` header and the views filter on it. Idempotent (`CREATE SOURCE IF NOT EXISTS` / drop-if exists if the image’s SQL requires it).

```sql
CREATE SOURCE IF NOT EXISTS src_auction_request_received
INCLUDE header 'protobuf_message' AS msg_type
WITH (
    connector = 'kafka',
    topic = 'telemetry-events',
    properties.bootstrap.server = 'redpanda:9092',
    scan.startup.mode = 'earliest'
) FORMAT PLAIN ENCODE PROTOBUF (
    message = 'org.bidon.telemetry.v1.AuctionRequestReceived',
    schema.registry = 'http://redpanda:8081',
    schema.registry.name.strategy = 'topic_record_name_strategy'
);

-- Same shape for:
--   src_auction_completed       message = 'org.bidon.telemetry.v1.AuctionCompleted'
--   src_dsp_response_received   message = 'org.bidon.telemetry.v1.DspResponseReceived'
```

Columns come from the descriptor: `envelope` is a struct (`(envelope).app_id`, `(envelope).auction_id`, `(envelope).event_ts`), enums decode as their value names (`OUTCOME_TIMEOUT`, `ERROR_CODE_NO_ADS_FOUND`), and proto3 unset strings arrive as `''`, not `NULL`. Header values are `bytea`. `price_floor` is the effective floor (a bid counts only when `price > price_floor`); `requested_price_floor` is what the SDK sent. `participant_count` equals the DSP sent and received counts for that auction (Amazon reports one received per request).

Typed views over the sources, then the funnel views (joins always `(app_id, auction_id)`):

```sql
CREATE MATERIALIZED VIEW IF NOT EXISTS auction_requests AS
SELECT (envelope).app_id AS app_id, (envelope).auction_id AS auction_id,
       (envelope).event_ts AS event_ts, requested_price_floor
FROM src_auction_request_received
WHERE convert_from(msg_type, 'utf8') = 'org.bidon.telemetry.v1.AuctionRequestReceived';

CREATE MATERIALIZED VIEW IF NOT EXISTS auctions_completed AS
SELECT (envelope).app_id AS app_id, (envelope).auction_id AS auction_id,
       (envelope).event_ts AS event_ts, winner_dsp, price, price_floor,
       participant_count, total_latency_ms, error_code
FROM src_auction_completed
WHERE convert_from(msg_type, 'utf8') = 'org.bidon.telemetry.v1.AuctionCompleted';

CREATE MATERIALIZED VIEW IF NOT EXISTS dsp_responses AS
SELECT (envelope).app_id AS app_id, (envelope).auction_id AS auction_id,
       (envelope).event_ts AS event_ts, dsp, outcome, http_status, latency_ms, price
FROM src_dsp_response_received
WHERE convert_from(msg_type, 'utf8') = 'org.bidon.telemetry.v1.DspResponseReceived';

CREATE MATERIALIZED VIEW IF NOT EXISTS auctions_in_flight AS
SELECT r.app_id, r.auction_id, r.event_ts AS started_ts
FROM auction_requests r
LEFT JOIN auctions_completed c
  ON r.app_id = c.app_id AND r.auction_id = c.auction_id
WHERE c.auction_id IS NULL;

-- Tumble 5 minutes on event_ts (ms → timestamptz). Adjust syntax to the pinned RW version.
CREATE MATERIALIZED VIEW IF NOT EXISTS dsp_outcomes_5m AS
SELECT window_start, dsp, outcome, COUNT(*) AS n
FROM TUMBLE(dsp_responses, to_timestamp(event_ts / 1000.0), INTERVAL '5 minutes')
GROUP BY window_start, dsp, outcome;

CREATE MATERIALIZED VIEW IF NOT EXISTS funnel_5m AS
SELECT
    window_start,
    COUNT(DISTINCT r.auction_id) AS requests,
    COUNT(DISTINCT c.auction_id) AS completed,
    COUNT(DISTINCT c.auction_id) FILTER (WHERE c.winner_dsp <> '') AS server_fills
FROM TUMBLE(auction_requests, to_timestamp(event_ts / 1000.0), INTERVAL '5 minutes') r
LEFT JOIN auctions_completed c
  ON r.app_id = c.app_id AND r.auction_id = c.auction_id
GROUP BY window_start;
```

Confirm `INCLUDE header`, `schema.registry.name.strategy` and `TUMBLE` / `to_timestamp` against the pinned RisingWave version’s docs and fix the syntax there — do not add extra views beyond the three typed ones and the three funnel views. Issue 5 will extend `funnel_5m`; keep the name stable.

Registry failures fail open in sdkapi: the record is produced as raw protobuf without the Confluent frame. A registry-backed source cannot decode those; treat them as dropped rows in the POC and check sdkapi logs for `schema registry register` if counts look low.

`CREATE SOURCE` may fail if the topic does not exist yet. Create `telemetry-events` explicitly (rpk in an init container, or document `rpk topic create`) because `AllowAutoTopicCreation` only fires on first produce. Registry subjects only exist after sdkapi’s first emit of each type, so run one auction before the SQL, or have the init container retry. Init order: redpanda healthy → topic create → one auction (or retry loop) → RW → SQL.

### How-to

Add `docs/telemetry-poc.md` (short):

- Console: `http://localhost:8080` → `telemetry-events`
- `psql -h localhost -p 4566 -U root -d dev` then `TABLE funnel_5m;`
- Grafana `http://localhost:3001` (set admin password in compose; do not commit a real secret — `admin`/`admin` is fine for dev)
- VT: how to open a `trace_id` from an event (Console decodes the protobuf via the registry; or `SELECT (envelope).trace_id` in `psql`)

### Accept / fail

- Compose up without errors on the new services.
- Stop `risingwave` / `victoriametrics` / `victoriatraces`; `curl` auction still 200 (use existing local auction fixtures / seed app key).
- Do not add Connect, Spaces, or DuckDB.
