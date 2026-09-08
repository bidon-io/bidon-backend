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

Blocked on issue 1 (topic + JSON shape). Useful after issue 2. Do not add `depends_on: risingwave` to `bidon-sdkapi`.

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

One wide source. Column list **must match** issue 1 JSON keys (snake_case). Idempotent (`CREATE SOURCE IF NOT EXISTS` / drop-if exists if the image’s SQL requires it).

```sql
CREATE SOURCE IF NOT EXISTS telemetry_events (
    event_id VARCHAR,
    event_name VARCHAR,
    event_ts BIGINT,
    schema_version VARCHAR,
    app_id BIGINT,
    auction_id VARCHAR,
    session_id VARCHAR,
    ad_type VARCHAR,
    ad_format VARCHAR,
    country VARCHAR,
    trace_id VARCHAR,
    sampling_rate DOUBLE,
    scope VARCHAR,
    dsp VARCHAR,
    outcome VARCHAR,
    http_status INT,
    latency_ms BIGINT,
    price DOUBLE,
    price_floor DOUBLE,
    winner_dsp VARCHAR,
    participant_count INT,
    total_latency_ms BIGINT,
    error_code VARCHAR,
    reject_reason VARCHAR
) WITH (
    connector = 'kafka',
    topic = 'telemetry-events',
    properties.bootstrap.server = 'redpanda:9092',
    scan.startup.mode = 'earliest'
) FORMAT PLAIN ENCODE JSON;
```

Views (joins always `(app_id, auction_id)`):

```sql
CREATE MATERIALIZED VIEW IF NOT EXISTS auctions_in_flight AS
SELECT r.app_id, r.auction_id, r.event_ts AS started_ts
FROM telemetry_events r
LEFT JOIN telemetry_events c
  ON c.event_name = 'auction_completed'
 AND r.app_id = c.app_id AND r.auction_id = c.auction_id
WHERE r.event_name = 'auction_request_received'
  AND c.auction_id IS NULL;

-- Tumble 5 minutes on event_ts (ms → timestamptz). Adjust syntax to the pinned RW version.
CREATE MATERIALIZED VIEW IF NOT EXISTS dsp_outcomes_5m AS
SELECT window_start, dsp, outcome, COUNT(*) AS n
FROM TUMBLE(
    (SELECT * FROM telemetry_events WHERE event_name = 'dsp_response_received'),
    to_timestamp(event_ts / 1000.0),
    INTERVAL '5 minutes'
)
GROUP BY window_start, dsp, outcome;

CREATE MATERIALIZED VIEW IF NOT EXISTS funnel_5m AS
SELECT
    window_start,
    COUNT(DISTINCT r.auction_id) AS requests,
    COUNT(DISTINCT c.auction_id) AS completed,
    COUNT(DISTINCT c.auction_id) FILTER (WHERE c.winner_dsp IS NOT NULL AND c.winner_dsp <> '') AS server_fills
FROM TUMBLE(
    (SELECT * FROM telemetry_events WHERE event_name = 'auction_request_received'),
    to_timestamp(event_ts / 1000.0),
    INTERVAL '5 minutes'
) r
LEFT JOIN telemetry_events c
  ON c.event_name = 'auction_completed'
 AND r.app_id = c.app_id AND r.auction_id = c.auction_id
GROUP BY window_start;
```

If `TUMBLE` / `to_timestamp` differs on the pinned version, fix to that version’s docs — do not add extra views. Issue 5 will extend `funnel_5m`; keep the name stable.

`CREATE SOURCE` may fail if the topic does not exist yet. Create `telemetry-events` explicitly (rpk in an init container, or document `rpk topic create`) because `AllowAutoTopicCreation` only fires on first produce. Init order: redpanda healthy → topic create → RW → SQL.

### How-to

Add `docs/telemetry-poc.md` (short):

- Console: `http://localhost:8080` → `telemetry-events`
- `psql -h localhost -p 4566 -U root -d dev` then `TABLE funnel_5m;`
- Grafana `http://localhost:3001` (set admin password in compose; do not commit a real secret — `admin`/`admin` is fine for dev)
- VT: how to open a `trace_id` from an event JSON

### Accept / fail

- Compose up without errors on the new services.
- Stop `risingwave` / `victoriametrics` / `victoriatraces`; `curl` auction still 200 (use existing local auction fixtures / seed app key).
- Do not add Connect, Spaces, or DuckDB.
