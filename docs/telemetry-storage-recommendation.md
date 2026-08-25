# Telemetry storage recommendation (historical)

**Status:** SUPERSEDED as a decision. **Do not implement the unified-ClickHouse path below.** Kept for option history.

**Current architecture:**

| Grain | Store | Doc |
| --- | --- | --- |
| A events | Parquet / Iceberg lake + DuckDB | [telemetry-events-store.md](./telemetry-events-store.md) |
| B traces | VictoriaTraces from day 1 | [telemetry-traces-store.md](./telemetry-traces-store.md) |
| C metrics | VictoriaMetrics from day 1 | [telemetry-metrics-store.md](./telemetry-metrics-store.md) |

Per-ad bytes and retention windows (validates the split): [telemetry-storage-sizing.md](./telemetry-storage-sizing.md).

Requirements: [telemetry-requirements.md](./telemetry-requirements.md). Spike architecture: [telemetry-m0-m1-backend-spike.md](./telemetry-m0-m1-backend-spike.md) §5. TRD: [TRD_BidOn_Telemetry.md](./TRD_BidOn_Telemetry.md) §6.

**Date of this historical note:** 2026-08-20 (settled-isolation note: 2026-08-21)  
**PRD:** [PRD_BidOn_Telemetry v1](./PRD_BidOn_Telemetry%20-%20v1.pdf)

---

## Why isolation was accepted

The rest of this file argues that one self-hosted ClickHouse is *good enough at all three grains* and cheaper in *people* than three products. That is still true as a capability claim. It was **not** adopted. Isolation was a grain decision, not a claim that three stores use less CPU or are simpler to run.

**Facts**

- The three jobs are different access patterns: catalog SQL with `sampling_rate` and `auction_id`; point-get of a span tree by `trace_id`; `rate()` every minute on **low-cardinality** labels. `auction_id` is a row/attribute, not a Prom series label.
- Alerts cannot wait on lake lag. Event SQL can. `/v2/show` does not query any of these stores; if a store is down, ads still serve.
- Self-hosted ClickHouse OSS is licence $0, same as VM/VT/DuckDB. The meter we refused for M0 is **ClickHouse Cloud / SaaS OLAP**, not a CH licence. Victoria and Grafana also have paid SKUs later.
- ClickHouse can do funnel SQL, `WHERE trace_id =`, and SQL `quantile`. VictoriaMetrics is PromQL/Alertmanager-native. VictoriaTraces is a GET-`trace_id` store. DuckDB runs **when someone queries** Parquet; it is not an always-on warehouse.
- Three stores means three query languages, three Grafana datasources, three “is it empty?” checks, plus otelcol and a Parquet sink. One CH means one process, one language, background merges, and a heavy query sharing the ingest node.
- At unmeasured Bidon M0 volume, **total RAM/CPU is not known to be smaller** for the split. Specialized engines are better-fit, not a guaranteed smaller machine. DuckDB-over-S3 will lose to CH if funnel SQL becomes daily/hourly and concurrent — that is events Phase 4 (CH **on the same lake**), not a redo of B or C.

**Accepted trade-offs**

| We took | We gave up |
| --- | --- |
| Each grain uses the engine that matches the query | One-box ops; one SQL pane |
| PromQL + Alertmanager for paging (C) | SQL alerts on the event/trace store |
| Event analytics off until query (A); traces GET in VT (B) | One backup, one on-call shape |
| Failure isolation: lake sink or DuckDB stall does not stop paging; VT down does not stop `/show` or VM; a funnel query cannot starve alerts | More daemons, more freshness stories, more ways to be “half up” |
| No Cloud OLAP meter in M0 | More people-time than one CH node (likely) |
| CH later only if DuckDB is the bottleneck, same files | Not using CH’s ability to hold A+B+C in one process |

Multiple failure domains are **both** a pro (blast radius) and a con (more to operate). That was accepted, not an accident of picking Victoria.

Do not read the historical sections below as “isolation was the worse option.” They are the case for unification, which we declined.

---

## Historical recommendation (rejected)

**Self-hosted ClickHouse (one node) is the store.** One SQL engine for product events, metric rollups, and traces. `otelcol-contrib` writes traces (and OTel metrics) into it; Redpanda `telemetry-events` lands in the same cluster (Kafka engine or a small consumer). Grafana optional. Sentry stays exceptions-only.

ClickHouse **Cloud** is what is expensive. A single Coolify/Compose instance is not in the same cost class. The earlier three-store design (Parquet + VictoriaMetrics + VictoriaTraces) was an answer to “do not buy a warehouse SaaS,” not to “ClickHouse cannot query this.” If cost was the objection, self-host one node.

Redpanda stays the ingest buffer and dual-emission path. It is not the query store.

## Prometheus vs `/metrics` vs PromQL

This repo **exposes** Prometheus *text* on `GET /metrics` (`echoprometheus`, OTel Prometheus exporter). That is a **client**. Nothing in this repo **scrapes or stores** those series. There is no Prometheus server unless Coolify/Grafana already added one outside the tree.

So:

- You do not “already have Prometheus” as a database.
- You have a scrape endpoint. Without a scraper + TSDB, those numbers vanish on process restart.
- **PromQL** is the query language of Prometheus-compatible TSDBs (Prometheus server, VictoriaMetrics, Mimir, …). Choosing VM means introducing that language *and* that store from zero.
- Spanmetrics is a collector *connector*: it turns spans into Prometheus-format series. Those series still need an **exporter destination** (VM, Prometheus, or ClickHouse). The connector is not storage.

## Collector

Correct: the collector is a pipeline, not a store. Receive OTLP → batch/sample → **export**. No exporter ⇒ spans are dropped (after optional spanmetrics, which also need an exporter). M0 therefore needs a real trace backend. Sentry does not count.

## Why three stores is the industry default — and why you can still use one

The industry splits because the access patterns differ:

| Job | Grain | Query |
| --- | --- | --- |
| Events (PRD catalog) | Row per event, `sampling_rate`, join `auction_id` | SQL funnels |
| Metrics | Time series, low cardinality | Alert p95 / rates |
| Traces | Span tree by `trace_id` | Fetch auction `abc`; latency per leg |

Nothing is *best* at all three. **ClickHouse is the only widely used self-hosted system that is good enough at all three for Bidon’s PRD** (funnel SQL, `WHERE trace_id =`, histograms via `quantile` / materialized views). That is the SigNoz/HyperDX/ClickStack model: three *tables*, one process.

VictoriaMetrics is better PromQL-native alerting. VictoriaTraces is a younger trace TSDB. Parquet+DuckDB is cheaper *if events are the only thing you store*. Together they are **three databases** (plus a sink). That is more RAM, more failure modes, and three query languages — the opposite of “least resistance” once you also need traces stored.

## Cost, honestly

| Option | Dollar cost at current scale | What you operate |
| --- | --- | --- |
| ClickHouse Cloud / Tinybird | High relative to Bidon M0 | Low ops, metered |
| **ClickHouse one node** (Compose/Coolify, 4–8 GiB) | Disk + that VM; licence $0 | One datastore, backups, disk |
| Parquet on R2 + DuckDB | Cheapest *events* | Sink + bucket; no always-on query |
| VM + VT + Parquet | Three sets of RAM/disk | Three products + sink |

If the CH objection is Cloud invoices: **do not buy Cloud.** One node next to Redpanda is the cost-controlled CH. It will likely be cheaper in *people* than VM+VT+lake, and similar in *machines* to VM+VT without solving events.

Promotion later is more replicas / ClickHouse Cloud, not a new schema. Same tables.

## What one ClickHouse instance actually does

```
bidon-sdkapi --OTLP--> otelcol-contrib --clickhouse exporter--> ClickHouse
     |                                                      ▲
     +-- telemetry-events --> Redpanda ---------------------+  (Kafka engine or consumer)
```

**Events.** MergeTree, partition `toDate(event_ts)`, `ORDER BY (event_name, auction_id, event_id)`. `sum(1 / sampling_rate)`. This is the PRD warehouse.

**Traces.** Separate table `ORDER BY (trace_id, span_id)` (OTel exporter schema). Store sampled auction trees from M0 for baselines. Grafana trace UI later; SQL/`trace_id` lookup from day one. Head-sample success at a **known rate**; do not tail-sample if you want unbiased p95 from traces. Unbiased p95: `quantile` on span duration (or a MV), not “only the slow traces we kept.”

**Metrics.** Two workable patterns — pick one:

1. **SQL-only (fewer systems):** OTel metrics + materialized views from spans/events (`dsp_latency_ms` by `demand_id`). Grafana alert on SQL. You never introduce PromQL.
2. **CH + PromQL:** still scrape `/metrics` into CH’s Prometheus-compatible endpoints *or* keep a tiny VM. Only do this if Grafana SQL alerts feel insufficient.

Recommend **(1)** for a small team that did not want three stores. `/metrics` can keep existing for process health; funnel/DSP SLOs live in CH.

**Spanmetrics** in the collector is optional if spans are in CH: you can aggregate in SQL. Spanmetrics is useful when the metric store is PromQL-native. With unified CH it is extra cardinality to manage, not a requirement.

## Trade-offs of unified ClickHouse

**Pros**

- One query language (SQL) for fill rate, DSP timeout, notice failures, “show spans for `auction_id`.”
- One backup, one disk, one on-call shape.
- Matches the query the PRD actually needs; traces are stored, not only rolled up.
- Files/export can still dump Parquet for icebox if you ever leave CH.
- Collector has a first-class ClickHouse exporter.

**Cons**

- ClickHouse is a sharp tool: merges, partitions, `ORDER BY` mistakes hurt. A small team can run **one node** if one person owns schema; they cannot ignore it.
- Trace waterfall UI is Grafana (or similar), not built-in. Storage does not require the UI.
- Alerting is weaker than Alertmanager+PromQL until you invest in Grafana unified alerting.
- A bad query can saturate the same node that is ingesting live auctions — isolate later (replicas) if that happens.
- You still need Redpanda (have it) and the collector (exporter target = CH).

## What you are not missing by skipping VM/VT (historical view)

This section is the **rejected** unification case. We did start with VM/VT: PromQL paging and a dedicated trace GET were chosen, not left in a back pocket.

VM/VT is a **vendor-shaped split**, not a requirement of the PRD. The historical argument: it is data-driven only if you already wanted PromQL and a dedicated trace TSDB; you do not have a Prometheus server today; one query surface points at CH. **We did want PromQL and a trace GET**, so that argument no longer applies.

Historical close: “Keep Victoria in the back pocket if SQL alerts fail you. Do not start there to avoid CH Cloud.” We started there for grain fit, not to dodge a CH licence.

## What Redpanda’s family actually covers

You already run the useful part: the **bus**. The rest of the family helps **land events**, not replace ClickHouse as the unified query store.

| Product | Helps | Does not |
| --- | --- | --- |
| **Redpanda (core)** | Buffer, replay, dual-emission, backpressure | Funnel SQL, traces, alerts |
| **Console** | Inspect topics | Partner reports |
| **Connect** (Benthos) | Sink `telemetry-events` → ClickHouse (or S3) without writing a consumer | OTLP from `bidon-sdkapi`; Connect’s `otlp_*` outputs *forward* OTel, they are not a trace database |
| **Iceberg topics** | Broker writes the topic as Iceberg/Parquet in object storage — the lake without a hand-rolled sink | Query engine (you still point DuckDB/Spark/CH/Athena at the table); **not** traces or PromQL |
| **Redpanda SQL** | Postgres-wire OLAP over live topic + Iceberg history | **Cloud BYOC**, not your Compose/Coolify node |
| **Tiered storage** | Cheaper long Kafka retention on S3 | Analytics schema |

Iceberg topics are the honest “use more Redpanda” move for **events**: enable Iceberg on `telemetry-events`, get Parquet in the bucket, query with DuckDB or let ClickHouse read Iceberg. Production Iceberg is aimed at BYOC/Enterprise + object storage; your current `docker-compose.dev.yml` Redpanda is a plain broker. Treat Iceberg as an upgrade path once Coolify Redpanda has a bucket, not as free on today’s container.

Still required outside the family: **OTLP collector + a trace (and metric) store**. Redpanda will not become VictoriaTraces or a PromTSDB. Putting spans on a Kafka topic and calling it traces leaves you scanning a log, not `GET trace_id`.

**Least resistance with what you have:** keep Redpanda as ingest; use **Connect → ClickHouse** (or Iceberg later) for events; collector → same ClickHouse for traces. That is “more Redpanda” where it is good (pipeline), one database where you query.

## Path (historical — do not follow)

1. **M0:** One ClickHouse; otelcol-contrib → CH traces (sampled); `telemetry-events` → CH; dual-emit old Kafka topics; Sentry traces off the auction path. No Cloud. No VM/VT. No Parquet sink unless you want a second copy for safety.
2. **Alerts:** Grafana (or CH `system.query_log` + a cron) on DSP p95 / ingest failures. Add `/metrics` scrape into CH only if process metrics are not already in OTel.
3. **M1:** Same tables, more event names. Auction/DSP spans (BE-M0-6) make the trace table match the TRD tree.
4. **Scale tripwire:** disk merge pressure, query vs ingest contention, or multi-month retention at tens of millions of rows/day → replica or Cloud. Schema stays.

## Stability (historical)

The rejected M0 was: events and sampled traces **queryable in one ClickHouse**, billing still on `/show`, old `ad-events` still flowing, collector exporting to CH not Sentry. **Settled M0 is the three-grain architecture** in the spike §5 / TRD §6 (not implemented in this repo yet).
