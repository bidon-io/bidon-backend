# Telemetry: requirements (clean slate)

**Status:** Draft — requirements only. Settled stores: grain A → [telemetry-events-store.md](./telemetry-events-store.md); grain B → [telemetry-traces-store.md](./telemetry-traces-store.md) (VictoriaTraces from day 1); grain C → [telemetry-metrics-store.md](./telemetry-metrics-store.md) (VictoriaMetrics from day 1).  
**Date:** 2026-08-26  
**Start here:** [telemetry-brief.md](./telemetry-brief.md)  
**PRD:** [PRD_BidOn_Telemetry v1](./PRD_BidOn_Telemetry%20-%20v1.pdf)  
**TRD:** [TRD_BidOn_Telemetry.md](./TRD_BidOn_Telemetry.md)  
**Code audit:** [telemetry-m0-m1-backend-spike.md](./telemetry-m0-m1-backend-spike.md)

Vendor notes from earlier discussion are *not* requirements. This document is what a store (or stores) must satisfy.

**Chosen:** grain A = Parquet lake (Iceberg when a catalog exists); grain B = VictoriaTraces; grain C = VictoriaMetrics. Per-unit bytes and volume tiers: [telemetry-storage-sizing.md](./telemetry-storage-sizing.md).

**This is a judgement call, not a conclusion these requirements force.** §6 permits one database if it passes both the §2.A funnel and the §2.B point-get, and a self-hosted single-node ClickHouse passes both — [telemetry-storage-recommendation.md](./telemetry-storage-recommendation.md) makes that case honestly and concedes the split is not obviously cheaper in machines or people. We chose the split for grain fit, PromQL-native paging, and failure isolation. Anyone re-reading these criteria will notice they do not discriminate; say "we preferred this" rather than "the requirements required it."

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
- Join client + server on **`(app_id, auction_id)`** — see G10.
- Never-sampled classes still present: errors, impressions, notices, funnel stages.

**This is the warehouse.** If a system cannot do those joins, it is not the event store.

**Who runs these queries.** J3/J4/J5 name BD, publisher support, solutions and finance. None of them run DuckDB over object storage. For M0/M1 these jobs are **engineer-mediated**: an engineer runs the saved SQL pack and returns numbers. That is a deliberate scope choice, not an oversight — but it is an expectation to set with those teams now, because "we have telemetry" will be heard as "I can self-serve." A query surface for non-engineers is an M2 conversation.

### B. Traces (per-auction tree)

One trace per auction; child span per DSP call; notice spans. `trace_id` rides on the **event envelope** and `auction_id` is a span attribute, so A→B is a point-get rather than a tag search.

**Queries (must work):**

- Fetch the tree for auction/trace X (point lookup).
- Quantiles of span duration by `demand_id` over a window (baselines). Head-sample rate **known and recorded** if traces are used for p95; otherwise p95 comes from C, not from “only slow traces we kept.”

**The sample rate has to be high enough that the tree is actually there.** This store exists to answer "show me the auction this partner is complaining about." At a 1% head sample it is absent 99% of the time, which fails the requirement while appearing to satisfy it. At current volume a trace tree is ~1.5 KiB and 14 days of *every* auction is under a gigabyte — so keep 100% until disk says otherwise ([traces store](./telemetry-traces-store.md#phases)).

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
| G5 | Dedupe on `event_id`. Redis at ingest is the M0 default; read-time dedupe on the lake is an equally valid implementation and the stated fallback when Redis RAM exceeds the lake's cost ([sizing §7](./telemetry-storage-sizing.md#7-tripwires)). Server-minted ids never need a dedupe entry. |
| G6 | Errors never sampled. Sampling rate on the event; server does not re-sample. |
| G7 | Kill switch via `/v2/config` (publisher). |
| G8 | Schema versioned; additive within a major. Version is **recorded, never a rejection reason**: the lake is schema-on-read, and PRD rationale 6 says old-SDK publishers persist for quarters and never reach zero. Rejecting them would bias the fill and reconciliation numbers J3/J4 exist to produce. |
| G9 | Collector/pipeline is not storage. Every signal in A/B/C has an **exporter destination** that retains it. |
| G10 | **`auction_id` and `session_id` are client-supplied on a public open-source SDK.** They are not globally unique and are not server-validated. Every join, dedupe key and partition predicate is scoped by `(app_id, …)`. A bare `auction_id` join is a correctness bug. |
| G11 | **Sampling is coherent per auction**, not independent per event: hash `(app_id, auction_id)`, keep or drop the whole diagnostic set. Funnel stages, errors, notices, impression and billing are never sampled at all. Without this, PRD success metric 1 (funnel completeness >98%) is unmeasurable by construction and J1 paths always have holes. |

---

## 4. Constraints (environment)

- Self-hosted (Coolify / Compose). Small team. Non-profit. Cost-sensitive: **SaaS OLAP meters** are the expensive thing; a single extra box is in play.
- Redpanda already exists and must keep receiving the old topics (G4).
- No Prometheus *server* today. No trace store today (Sentry only).
- **This is a first iteration.** The design point is ~10k DAU with headroom checked to 1M, each component carrying a written tripwire ([sizing §7](./telemetry-storage-sizing.md#7-tripwires)). It is not sized for 10M DAU and does not need to be.
- Volume unmeasured. The PRD's 20–40× events/impression is a hypothesis; modelling the full PRD catalog gives **~55 raw events per auction**, above that band, which is why sampling is load-bearing rather than optional. Raw emit sizes the bus, the client queue and ingest; *stored* sizes the lake. See [telemetry-storage-sizing.md](./telemetry-storage-sizing.md).
- Open source / public SDK: collection story must stay minimal, gated, and **allow-listed** (PRD rationale 5 — over-collection here cannot be quietly fixed).

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

If you pick **two** (events vs traces), that is allowed. Three is allowed.

**These criteria do not pick a winner, and should not be presented as if they do.** A self-hosted single-node ClickHouse passes §2.A and §2.B. What (4) genuinely rules out is *lake-only*: a Parquet sink flushing on an interval cannot back a 5-minute alert `for:` window. That argues for having a metrics path — it does not argue against ClickHouse, which pairs with Grafana alerting or a small VM. We chose three stores; §7 is a fit test, not a proof.

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
- Direct-mode events per auction (measure; model says ~55 raw).  
- Whether `/v2/stats` `ad_units[]` is attempt-ordered (fill metrics).  
- Impression definition vs OM (meaning of `ad_impression`).  
- Legal retention for G3, which is **three questions, not one**: (a) the statutory accounting window for the *ledger* (commonly 6–10 y — satisfied by aggregates, not raw events); (b) the **contractual** dispute and audit window in the MAX certification terms and DSP agreements, which is what actually binds impression-level retention and is currently unchecked; (c) whether `session_id` + device + bundle is pseudonymous personal data. A "yes" on (c) makes long retention a liability rather than prudence, and makes row-level delete an M0 requirement — which promotes Iceberg out of Phase 2.

Until those, capacity and monthly $ stay ranges from [telemetry-storage-sizing.md](./telemetry-storage-sizing.md), not a bill.

### Gaps in PRD v1 (raised with the PRD owner)

- **"Note 2" does not exist.** All five regulation fields (`gdpr_applies`, `has_tcf_string`, `us_privacy`, `coppa`, `lmt`) route to it; the document ends mid-sentence at `Note 2(`. G3 and the COPPA reduced field set are built on it.
- **The ad-path latency risk row is corrupted** — its mitigation cell repeats the M0 scope text and its impact reads "N/A". Combined with the "N/A" latency success target, the PRD currently states no latency gate for the one risk G1 exists to address.
- `sampling_rate` is required by the PRD's sampling section but absent from the PRD envelope table.
- "Telemetry loss rate < 1% (`telemetry_dropped` ÷ emitted)" is a *client* counter with no transport in the catalog. The server's `telemetry_dropped_total` measures server-side drops — a different quantity.

---

## 9. Events-only store (grain A)

See [telemetry-events-store.md](./telemetry-events-store.md): **phased** — Parquet on S3 + DuckDB (Phase 1); Iceberg via OSS writer + REST catalog (Phase 2); Athena/Trino if needed (Phase 3); ClickHouse reads the **same** lake (Phase 4). No Redpanda Enterprise.

---

## 10. Traces store (grain B)

See [telemetry-traces-store.md](./telemetry-traces-store.md): **VictoriaTraces** from day 1. OTLP via otelcol-contrib. GET by `trace_id`. Sentry exceptions only. Unbiased p95 / paging is grain C (spanmetrics), not this store.

---

## 11. Metrics store (grain C)

See [telemetry-metrics-store.md](./telemetry-metrics-store.md): **VictoriaMetrics** from day 1. Scrape `/metrics` + OTel Prometheus exporter. vmalert + Alertmanager. No `auction_id` on series.
