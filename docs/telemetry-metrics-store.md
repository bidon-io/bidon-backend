# Operational metrics: VictoriaMetrics from day 1

**Scope:** grain C only ([telemetry-requirements.md](./telemetry-requirements.md) §2.C). Grain A: [telemetry-events-store.md](./telemetry-events-store.md). Grain B: [telemetry-traces-store.md](./telemetry-traces-store.md) (VictoriaTraces from day 1). Spike architecture: [telemetry-m0-m1-backend-spike.md](./telemetry-m0-m1-backend-spike.md) §5.

**Decision:** scrape existing `GET /metrics` (and later the OTel Prometheus exporter) into **single-node VictoriaMetrics**. **vmalert** evaluates rules; **Alertmanager** pages. Grafana uses the Prometheus datasource pointed at VM. **`vmbackup` to S3** when local retention is not enough. **VM cluster** only if scrape volume hurts.

These are **phased**. Each phase keeps the produce path: in-process Prometheus *client* on `/metrics`. We are not running a Prometheus *server*.

`/v2/show` does not query VM. If VM is down, ads still serve; you just do not page (G1/G2).

---

## Argument

Alerts need a process that answers PromQL every minute. Events can sit unread on S3; **this grain cannot**. The lake-like part is cheap local retention (VM compression) plus optional cold copies on object storage — not a metered SaaS TSDB, not Parquet+DuckDB `rate()`.

The repo already **exposes** Prometheus text. VM is Prometheus-compatible for scrape, Grafana, and (via vmalert) alerting rules. MetricsQL is a PromQL superset; typical dashboards work. `rate`/`increase` are intentionally slightly different from Prometheus — acceptable for new rules written against VM.

Do not put `auction_id` (or any high-cardinality id) on a series. Per-auction latency lives in **traces (B)** or **events (A)**.

---

## Same model as the event lake

| Event lake (A) | Metrics (C) |
| --- | --- |
| Produce JSON, do not await the warehouse | Increment counters in-process; do not await VM |
| Redpanda is the durable log | VM local storage is the durable log for the hot window |
| Parquet on S3 is the cheap retain | **`vmbackup` (later cluster VM)** on the same bucket family, `metrics/` prefix |
| DuckDB when a human runs SQL | **MetricsQL** when vmalert or Grafana runs a range query |
| Iceberg when writers/engines need a catalog | VM cluster when one node is not enough |
| ClickHouse later as a hot accelerator | Do not add CH *only* for metrics |

---

## How series are born (produce)

The app never blocks on VM.

| Source | What it is | Labels (keep tiny) |
| --- | --- | --- |
| Existing `/metrics` | Process, HTTP, Go runtime | `job`, `instance`, maybe `path` **not** auction |
| New counters on `/v2/show` | Impression handler success/fail, BURL enqueue fail | `job`, `result` |
| OTel **spanmetrics** (from grain B) | `dsp_request_duration`, timeout count | `dsp`, `operation` — **never** `trace_id` / `auction_id` |
| Collector scrape fail | Pipeline health | `job` |

Do **not** page off the event lake until sink freshness is inside the alert `for:` window. Until that SLO exists, use scrape.

---

## Phases

### Phase 1 — single-node VM (M0, day 1)

```
bidon-sdkapi GET /metrics  →  VictoriaMetrics (single, scrape)
otelcol Prometheus exporter ↗
                              →  vmalert  →  Alertmanager
                              →  Grafana (Prometheus datasource → VM)
```

- Compose/Coolify sidecar. Retention **30–90d** is fine on one disk at low cardinality; no need to start at 15d.
- Scrape config can live on VM itself for a handful of targets. Split out **vmagent** only if scrape fan-out grows.
- Rules: Prometheus-style YAML in **vmalert**. Alertmanager for routing.
- Query language: MetricsQL. Write new alerts against VM; do not assume Prometheus `rate()` extrapolation.

**Good enough** to prove C: “page if DSP timeout rate is bad for N minutes.”

### Phase 2 — cold copy on object storage (when local disk is not enough)

`vmbackup` to `s3://…/metrics/` (same bucket family as events, different prefix). This is backup/restore and capacity relief, not a live PromQL-on-S3 lake. Alerts still hit **Phase 1**.

Not Thanos: VM is not Prometheus TSDB blocks. Not Mimir. Not Iceberg of Prom samples.

### Phase 3 — VM cluster (only if scrape volume hurts)

vminsert / vmselect / vmstorage. Skip until one node is the problem. Do not stand up ClickHouse *only* for metrics.

---

## What we are not doing

| Skip | Why |
| --- | --- |
| Prometheus *server* | VM from day 1; `/metrics` client stays |
| Thanos sidecar | Wrong on-disk format |
| Grafana Cloud / Datadog / Chronosphere | Metered SaaS |
| Iceberg tables of Prom samples | Wrong query language |
| Kafka topic of every scrape sample | Redpanda is the **event** log. VM has its own WAL |
| High-cardinality labels | `auction_id`, `creative_id`, raw endpoint URLs |
| Lake-derived alerts before sink SLO | Freshness unknown; pages would lie |

---

## Relationship to A and B

- **A (events):** historical fill rate — SQL on Parquet. Not paging.
- **B (traces):** p95 / timeout on **this auction** — span store. Spanmetrics **rolls B up into C** without putting ids on series.
- **C (this doc):** “is it bad *right now* for this DSP / this job.”

Three grains can share **object storage** (prefixes) and **one Grafana**. They do not share one table format.

---

## Compatibility (so we do not get surprised)

VM is **Prometheus-compatible**, not Prometheus:

- `/metrics` scrape, Grafana Prometheus datasource, Alertmanager: yes.
- Recording/alerting rules: **vmalert**, not in-process Prometheus.
- MetricsQL vs PromQL: `rate`/`increase` differ on purpose (no extrapolation; include sample before the window).
- Migration off Prometheus disk: N/A — we never run a Prometheus server.

---

## Decision

| Item | Choice |
| --- | --- |
| Hot store (alerts) | Single-node **VictoriaMetrics**, scrape `/metrics` + OTel Prometheus exporter |
| Rules / paging | **vmalert** + **Alertmanager** |
| Dashboards | Grafana → Prometheus datasource → VM |
| Cold store (optional) | **`vmbackup`** to S3 `metrics/` prefix |
| Scale-up | VM cluster, not CH-for-metrics |
| Not Iceberg | Metrics are MetricsQL, not DuckDB |
| Not the event topic | Do not dual-write samples to `telemetry-events` |
| Labels | Low cardinality only; no `auction_id` |
| Lake alerts | Only after sink freshness ≤ alert `for` window |

Owners still needed: who runs the VM instance, vmalert rules, alert routing, and (later) the metrics prefix on the bucket.
