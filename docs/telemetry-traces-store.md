# Operational traces: VictoriaTraces from day 1

**Scope:** grain B only ([telemetry-requirements.md](./telemetry-requirements.md) §2.B). Grain A: [telemetry-events-store.md](./telemetry-events-store.md). Grain C: [telemetry-metrics-store.md](./telemetry-metrics-store.md). Spike architecture: [telemetry-m0-m1-backend-spike.md](./telemetry-m0-m1-backend-spike.md) §5.

**Decision:** export OTLP from `bidon-sdkapi` via **otelcol-contrib** into **single-node VictoriaTraces**. GET the tree by `trace_id` (Jaeger API; Tempo HTTP API when useful). Grafana later, optional. **Spanmetrics → VictoriaMetrics** for unbiased p95 and paging. **Sentry stays exceptions only.** Cluster VT only if one node hurts. Backup/`vmbackup`-style copy to S3 when local disk is not enough — VT is not a live Parquet lake.

These are **phased**. Each phase keeps the produce path: OTel SDK, never await from the ad path.

`/v2/show` does not query VT. If VT is down, ads still serve; you just cannot load the tree (G1/G2).

---

## Argument

A trace is a **point lookup** (auction X’s DSP legs), not funnel SQL and not a Prom series. Events can sit unread on S3; traces can sit unread on **local VT disk** until someone GETs them. Paging cannot wait on that GET — that is why p95 lives in **VictoriaMetrics**.

We already run **VictoriaMetrics** for grain C. VictoriaTraces is the same-family ops shape: one vendor, OTLP in, local storage, Grafana datasources later, cluster when needed. It is **not** Tempo’s Parquet-on-S3 model. That trade was explicit: family over lake physics.

Do not put `auction_id` on a **metric** label. On a span it is an **attribute** so events and traces join.

---

## Same model as events / metrics

| Event lake (A) | Metrics (C) | Traces (B) |
| --- | --- | --- |
| Produce JSON, do not await the warehouse | Increment counters, do not await VM | Export OTLP, do not await VT |
| Redpanda is the durable log | VM local storage is the hot WAL | VT local storage is the retain |
| Parquet on S3 is the cheap retain | `vmbackup` to S3 later | **Backup to S3 later** (not live GET-from-Parquet) |
| DuckDB when a human runs SQL | MetricsQL when vmalert runs | **GET `trace_id`** when a human debugs |
| Iceberg when engines need a catalog | VM cluster when one node hurts | **VT cluster** when one node hurts |
| ClickHouse later as SQL accelerator | Do not add CH for metrics | Do not add CH *only* for traces |

**What transfers from events:** debugging is on-demand. No always-on PromQL.

**What does not transfer:** DuckDB `WHERE trace_id =` on the event lake is a full scan. VT exists so GET is an index lookup. Redpanda is the **event** log; OTLP is the trace wire.

---

## How traces are born (produce)

The auction path never blocks on VT.

```
bidon-sdkapi (OTel SDK) → otelcol-contrib
                            → OTLP exporter → VictoriaTraces
                            → spanmetrics connector → VictoriaMetrics
```

| Span | Parent | Attributes (keep useful, not huge) |
| --- | --- | --- |
| `auction.run` | HTTP `/v2/auction` | `auction_id`, `app_id`, `ad_type` |
| `auction.dsp.{demand_id}` | `auction.run` | `demand_id`, `http.status`, `outcome` |
| `notification.send` | `auction.run` or `/v2/show` | `notice_type`, `demand_id`, `http.status` |

Head-sample at a **known** rate (or errors + a fraction of successes). Record the rate. Tail sampling (“keep only slow”) is allowed for extra debug volume; then **do not** use this store for p95.

Collector is a pipe. No exporter destination ⇒ spans are dropped (G9).

**Already in the repo:** `ConfigureOTel()` sends spans to **Sentry**. Auction traces must not dual-write there.

---

## Phases

### Phase 1 — single-node VT (M0, day 1)

```
otelcol → VictoriaTraces (single)
            → GET /select/jaeger/api/traces/{trace_id}
            → Grafana later (Jaeger or Tempo datasource → VT)
```

- Compose/Coolify sidecar. Local disk. Retention sized to sampled auction trees, not every span forever.
- Query: Jaeger JSON API from day one. Tempo HTTP API is experimental on VT — fine for Grafana later, not a blocker.
- Prove B: load auction X’s tree via `trace_id` (and `auction_id` as a tag/attribute filter). No UI required.

### Phase 2 — cold copy on object storage (when local disk is not enough)

Backup VT data to `s3://…/traces/` (same bucket family as events, different prefix). Disaster recovery / capacity, not a live trace-lake. GETs still hit **Phase 1**.

Not Tempo S3 blocks. Not Iceberg of spans. Not spans on `telemetry-events`.

### Phase 3 — VT cluster (only if ingest or lookup hurts)

`vtinsert` / `vtselect` / `vtstorage`. Skip until one node is the problem.

---

## What we are not doing

| Skip | Why |
| --- | --- |
| Prometheus / Tempo *server* as the trace store | VT from day 1; same family as VM |
| Sentry as the auction trace backend | Not queryable for reports; exceptions only |
| Spans on `telemetry-events` / DuckDB `WHERE trace_id` | Wrong grain; full scan |
| Grafana Cloud Tempo / Honeycomb / Datadog | Metered SaaS |
| Jaeger + Elasticsearch / Cassandra | Index database you do not want to run |
| ClickHouse traces table | Unified-CH was rejected for A and C |
| Tempo metrics-generator **and** collector spanmetrics | One rollup into VM; collector spanmetrics is enough |
| High-cardinality Prom labels | `auction_id` stays a span attribute |
| Tail-sample-only traces used as p95 | Biased; p95 lives in VM |

---

## Relationship to A and C

- **A:** “what was fill rate last week” — SQL on Parquet. Join VT via `auction_id` when a row is interesting.
- **B (this doc):** “show me auction X’s DSP legs.”
- **C:** “is DSP timeout rate bad *right now*” — VM. Spanmetrics rolls B → C without ids on series.

Three grains can share **one Grafana**. Events share a bucket with optional VT backups (`events/`, `metrics/`, `traces/`). They do not share one table format.

---

## Decision

| Item | Choice |
| --- | --- |
| Store | Single-node **VictoriaTraces**, OTLP from otelcol-contrib |
| Lookup | Jaeger API GET by `trace_id`; `auction_id` as span attribute |
| p95 / paging | **VictoriaMetrics** via spanmetrics, not this store |
| UI | Optional; API GET is M0 |
| Cold copy (optional) | Backup to S3 `traces/` prefix |
| Scale-up | VT cluster, not CH-for-traces, not Tempo |
| Not the event topic | Do not write spans to `telemetry-events` |
| Not Sentry | Exceptions only |
| Sample | Known head rate; tail-sample only for extra debug volume |

Owners still needed: who runs the VT instance, head-sample policy, and Sentry config so auction trees stop going there.
