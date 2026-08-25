# Telemetry: requirements (clean slate)

**Status:** Draft — requirements only. Settled stores: grain A → [telemetry-events-store.md](./telemetry-events-store.md); grain B → [telemetry-traces-store.md](./telemetry-traces-store.md) (VictoriaTraces from day 1); grain C → [telemetry-metrics-store.md](./telemetry-metrics-store.md) (VictoriaMetrics from day 1).  
**Date:** 2026-08-21  
**PRD:** [PRD_BidOn_Telemetry v1](./PRD_BidOn_Telemetry%20-%20v1.pdf)  
**TRD:** [TRD_BidOn_Telemetry.md](./TRD_BidOn_Telemetry.md)  
**Code audit:** [telemetry-m0-m1-backend-spike.md](./telemetry-m0-m1-backend-spike.md)

Vendor notes from earlier discussion are *not* requirements. This document is what a store (or stores) must satisfy.

**Settled (choice):** grain A = Parquet/Iceberg lake; grain B = VictoriaTraces; grain C = VictoriaMetrics. Three stores because alerts cannot wait on lake lag, and a trace is a point-get, not funnel SQL. Unified ClickHouse is rejected ([telemetry-storage-recommendation.md](./telemetry-storage-recommendation.md) is history only). Per-ad bytes and aggregation windows: [telemetry-storage-sizing.md](./telemetry-storage-sizing.md).

---

## 1. Why this exists (jobs)

The PRD is delivery evidence, not a dashboard product.

| # | Job | Who | If we fail |
| --- | --- | --- | --- |
| J1 | Reconstruct an ad request’s path from init/load → terminal outcome | Eng, support | Cannot tell Bidon vs publisher vs DSP |
| J2 | Attribute a failure to a domain + stable code (foreign code preserved) | Eng, partners | Every escalation is a reproduction |
| J3 | Quote fill, time-to-fill, render rate, per-source load, DSP timeout | Pub support, BD | No proving-ground numbers for MAX |
| J4 | Reconcile Bidon impressions vs mediator / DSP billing | Cert, finance | Accept AppLovin’s numbers |
| J5 | Show per-DSP loss and render (acquisition + DSP ops) | BD, solutions | No commercial evidence |
| J6 | See where the latency budget went (token / auction / DSP / notices) | Server eng | Timeouts are vibes |
| J7 | Do all of the above without slowing the ad path | Everyone | Telemetry is net-negative |

M0/M1 only: Android-first, direct mode (+ server pieces reused in later modes). M2/M3 events, OM, and a reporting UI are out of this slice.

---

## 2. Three kinds of data (do not collapse them in the spec)

They may share *infrastructure*. They must not share *grain*. Mixing grains is how Sentry and `/metrics` failed as the answer.

### A. Product events (the catalog)

One row per thing that happened, PRD envelope (`event_id`, `event_name`, `schema_version`, `auction_id`, `session_id`, `sampling_rate`, …).

Server must produce at least (M1):  
`auction_request_received`, `dsp_request_sent`, `dsp_response_received`, `dsp_response_rejected`, `auction_completed`, `win|billing|loss_notice_sent`, `notice_delivery_failed`.  
`ad_impression` / billing are **derived from `/v2/show`**, not from the SDK batch queue.

Client events arrive via a new ingest (`/v2/telemetry`) and/or existing domain routes during dual-emission.

**Queries (must work):**

- Funnel: count `ad_request_started` → `ad_filled` → `ad_impression` → `billing_notice_sent`, **corrected by** `1 / sampling_rate`.
- `GROUP BY dsp_id` / `demand_source` / `error_code`.
- Join client + server on `auction_id` (today SDK-minted).
- Never-sampled classes still present: errors, impressions, notices.

**This is the warehouse.** If a system cannot do those joins, it is not the event store.

### B. Traces (per-auction tree)

One trace per auction; child span per DSP call; notice spans. Attributes include `auction_id` so events and traces join.

**Queries (must work):**

- Fetch the tree for auction/trace X (point lookup).
- Quantiles of span duration by `demand_id` over a window (baselines). Head-sample rate **known and recorded** if traces are used for p95; otherwise p95 comes from C, not from “only slow traces we kept.”

**UI is not an M0 requirement.** Storage is. Sentry is not this store (not actionable for reports).

**This is not the funnel.** You do not define fill rate from spans.

### C. Operational signals (alerts)

Low-cardinality time series: ingest 5xx, Kafka produce fail, DSP timeout *rate*, notice failure *rate*, process health.

**Queries (must work):** “page if X is bad for N minutes.” Grain is *not* `auction_id`.

Today `GET /metrics` **exposes** Prometheus text. Nothing in this repo **stores** it. A scrape endpoint is not a requirement satisfied.

---

## 3. Guarantees (non-negotiable)

| ID | Requirement |
| --- | --- |
| G1 | Telemetry must not delay or fail an ad request (enqueue, do not await), except G2. |
| G2 | Impression + BURL stay on `POST /v2/show`. Warehouse row may lag; billing must not. |
| G3 | New stream: no PII (no IP/IFA/IDFV). `country` from MaxMind. COPPA reduced set. |
| G4 | Dual-emit existing `ad-events` / `notification-events` until that consumer is named and migrated. New catalog is a **new** stream. |
| G5 | Dedupe on `event_id` at ingest (Redis 72h is the TRD default). |
| G6 | Errors never sampled. Sampling rate on the event; server does not re-sample. |
| G7 | Kill switch via `/v2/config` (publisher). |
| G8 | Schema versioned; additive within a major. |
| G9 | Collector/pipeline is not storage. Every signal in A/B/C has an **exporter destination** that retains it. |

---

## 4. Constraints (environment)

- Self-hosted (Coolify / Compose). Small team. Cost-sensitive: **SaaS OLAP meters** are the expensive thing; a single extra box is in play.
- Redpanda already exists and must keep receiving the old topics (G4).
- No Prometheus *server* today. No trace store today (Sentry only).
- Volume unmeasured; PRD 20–40× events/impression is a hypothesis to *measure*, not a capacity spec. Planning model (1 DAU × 1 served ad, then scale): [telemetry-storage-sizing.md](./telemetry-storage-sizing.md).
- Open source / public SDK: collection story must stay minimal and gated (PRD).

---

## 5. Explicitly not required in M0/M1

- Publisher-facing dashboards, alerting *product*, partner portal.
- Trace waterfall UI (store without Grafana is fine).
- M2/M3 event catalog, OM definition (blocks `ad_impression` *meaning*, not `/show` as channel).
- Cutting over old Kafka topics.
- Server-minted `auction_id` (joint SDK change; correlate with what exists).
- Replacing Redpanda as the ingest log.

---

## 6. What “a solution” has to be

Minimum moving parts that satisfy §2–3:

1. **A log** — already Redpanda — so producers stay fire-and-forget and G4 holds.  
2. **A retained event table** — SQL-capable (or equivalent) for §2.A queries.  
3. **A retained trace table or TSDB** — point-get by `trace_id` + duration quantiles for §2.B.  
4. **An alert path** — may be rollups of (2) or (3), or a scrape of `/metrics`. Must not use `auction_id` as a series label.  
5. **A collector** (or equivalent) that **exports** OTLP to (3) and optionally (4). Not optional if we want B.

(2) and (3) **may be the same database** (different tables). (4) may be derived from (2)/(3) so you do not buy a TSDB. That is a *store-count* optimisation, not a requirement to merge grains.

If you pick **one** database, it must still pass the §2.A funnel queries *and* §2.B point-get. If it fails either, it is not unified — it is the wrong one box.

If you pick **two** (events vs traces), that is allowed. Three (events + traces + dedicated metrics TSDB) is justified when (4) cannot be derived from the lake inside the alert `for:` window — which is the Bidon M0 situation.

---

## 7. How to judge a candidate (sniff test)

Ask, in order:

1. Can I write the fill-rate SQL (or equivalent) with `sampling_rate`?  
2. Can I load auction X’s span tree after a week?  
3. Can I alert on DSP timeout rate without `auction_id` on the series?  
4. Does `/show` billing still work if this store is down?  
5. Can a small team back it up and notice it is empty (freshness)?  
6. Is the cost **Cloud meter** or **one node**? Reject the former for M0 unless volume forces it.

Anything that only answers 3 (classic PromQL TSDB) or only answers 2 (VictoriaTraces / Tempo) is a *component*, not the whole solution.

---

## 8. Open facts (not product choices)

- Who consumes `ad-events` today.  
- Direct-mode events per impression (measure).  
- Whether `/v2/stats` `ad_units[]` is attempt-ordered (fill metrics).  
- Impression definition vs OM (meaning of `ad_impression`).  
- Legal retention for G3.

Until those, capacity and monthly $ stay ranges from [telemetry-storage-sizing.md](./telemetry-storage-sizing.md), not a bill.

---

## 9. Events-only store (grain A)

See [telemetry-events-store.md](./telemetry-events-store.md): **phased** — Parquet on S3 + DuckDB (Phase 1); Iceberg via OSS writer + REST catalog (Phase 2); Athena/Trino if needed (Phase 3); ClickHouse reads the **same** lake (Phase 4). No Redpanda Enterprise.

---

## 10. Traces store (grain B)

See [telemetry-traces-store.md](./telemetry-traces-store.md): **VictoriaTraces** from day 1. OTLP via otelcol-contrib. GET by `trace_id`. Sentry exceptions only. Unbiased p95 / paging is grain C (spanmetrics), not this store.

---

## 11. Metrics store (grain C)

See [telemetry-metrics-store.md](./telemetry-metrics-store.md): **VictoriaMetrics** from day 1. Scrape `/metrics` + OTel Prometheus exporter. vmalert + Alertmanager. No `auction_id` on series.
