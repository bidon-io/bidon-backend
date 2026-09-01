# Technical Requirements Document — Telemetry event storage

**Parquet on DigitalOcean Spaces**

**Status:** Draft  
**Date:** 2026-09-01  
**This TRD specifies:** the events warehouse. One row per product event, retained as Parquet on DigitalOcean Spaces, flushed every 10 minutes.  
**This TRD does not specify:** how events are produced, ingested, sampled, or privacy-filtered; traces; metrics.

| | |
| --- | --- |
| Why the warehouse exists | [PRD v1](./PRD_BidOn_Telemetry%20-%20v1.pdf) — Goals 1–4; Part II “raw events to a columnar warehouse for funnel joins” |
| What the warehouse must answer | [telemetry-requirements.md](./telemetry-requirements.md) §2.A (jobs J3, J4, J5) |
| Why Parquet on object storage | [telemetry-events-store.md](./telemetry-events-store.md) |
| Numbers | [telemetry-storage-sizing.md](./telemetry-storage-sizing.md) |
| Producers, ingest, dual-emit | [telemetry-m0-m1-backend-spike.md](./telemetry-m0-m1-backend-spike.md) |
| Traces | [telemetry-traces-store.md](./telemetry-traces-store.md) |
| Metrics | [telemetry-metrics-store.md](./telemetry-metrics-store.md) |

No production instrumentation ships from this draft. The lake can exist empty; producers are a different piece of work.

---

## 1. Scope

### In

- A DigitalOcean Spaces bucket (S3 API) for telemetry events.
- A **custom** Parquet writer (new Go binary in this repo) that consumes Redpanda `telemetry-events`. Not a licensed Redpanda or Confluent sink.
- Hive layout, 10-minute flush, compression.
- Lifecycle retention on the two raw classes, plus a daily aggregate prefix.
- A DuckDB SQL pack an engineer can run against the bucket.
- A freshness alert on the sink.

### Out

| Topic | Where it lives instead |
| --- | --- |
| `POST /v2/telemetry`, batch size, gzip, 4xx vs 5xx | Spike BE-M0-3 |
| `/v2/config` telemetry block, kill switch, sample rate | Spike BE-M0-4 |
| Envelope types, `schema_version`, error-code table | Spike BE-M0-1; BID-46 |
| Redis `event_id` dedupe | Requirements G5; spike BE-M0-3 |
| Sampling policy (coherent per auction) | Requirements G11; sizing §3 |
| Privacy allow-list, COPPA field set, `raw_request` exclusion | Requirements G3; spike BE-M0-5 |
| Dual-emit of `ad-events` / `notification-events` | Requirements G4; spike §4 |
| `/v2/show` as billing channel | Requirements G2; spike BE-M1-4 |
| OTel traces, VictoriaTraces | [telemetry-traces-store.md](./telemetry-traces-store.md) |
| `/metrics` scrape, VictoriaMetrics, paging | [telemetry-metrics-store.md](./telemetry-metrics-store.md) |
| Redpanda Enterprise Iceberg topics, Redpanda Connect Iceberg/S3, Confluent Kafka Connect S3 | Licensed / existing JSON sink — not this store |
| Publisher-facing BI, a second query engine, a table catalog | Out |

### Upstream assumptions (not built here)

The sink reads JSON envelopes already on `telemetry-events`. Those envelopes are assumed to already satisfy:

- PRD common envelope (`event_id`, `event_name`, `event_ts`, `schema_version`, `auction_id`, `session_id`, `sampling_rate`, `app_id`, …).
- No PII (G3). `country` already derived; no IP / IFA / IDFV / city / `raw_request` / `raw_response`.
- `sampling_rate` already stamped at emit. The sink does not re-sample and does not drop by rate.

If the topic is empty, the lake is empty. That is acceptable for this TRD.

---

## 2. Why this store

The PRD’s jobs that need a warehouse are catalog SQL, not point-gets and not alerts:

| Job | Query the lake must support |
| --- | --- |
| J3 | Fill rate, time-to-fill, render rate, per-source load — `sum(1 / sampling_rate)`, `GROUP BY` demand |
| J4 | Reconcile Bidon impressions vs mediator / DSP billing |
| J5 | Per-DSP loss and render |

Requirements §2.A: if a system cannot do those joins, it is not the event store. Minutes of lag are fine. Paging is not this store’s job.

The PRD also says telemetry must not delay or fail an ad request. The lake is downstream of Redpanda. If the sink or Spaces is down, ads still serve (G1/G2). `/v2/show` produces events onto the bus; it does not query this bucket.

**Who runs the SQL.** J3/J4/J5 name BD, publisher support, solutions and finance. For this slice those jobs are engineer-mediated: someone runs the saved SQL pack and returns numbers. A query surface for non-engineers is not in this TRD.

---

## 3. Pipeline

```
Redpanda telemetry-events (JSON, already produced)
        │
        ▼
Parquet sink (custom Go, this repo)  ── flush every 10 min, or 64 MiB uncompressed, whichever first
        │
        ▼
DigitalOcean Spaces
  events/dt=YYYY-MM-DD/event_name=…/*.parquet          ← raw
  aggregates/dt=YYYY-MM-DD/*.parquet                   ← daily rollup
        │
        ▼
DuckDB  (read_parquet, on demand)
```

JSON on the topic is the **bus**. Parquet on Spaces is the **store**. Do not land JSON-on-S3 for this catalog.

The writer is a **custom Go consumer** we build. It is not:

- Redpanda Iceberg topics (`redpanda.iceberg.mode`) or any other Redpanda Enterprise sink
- Redpanda Connect / Benthos Iceberg or S3 output
- Confluent Kafka Connect S3 — including the existing `docker/kafka-connect` connector, which writes **JSON** of `ad-events`. Do not point that connector at `telemetry-events`, and do not extend it.

---

## 4. Requirements

### 4.1 Bucket

| Item | Requirement |
| --- | --- |
| Service | DigitalOcean **Spaces** (S3 API) |
| Isolation | Dedicated telemetry bucket, or a dedicated prefix on a bucket that is **not** the Postgres-backup bucket (`infra` currently creates `${name_prefix}-backups-*`). Backup lifecycle must not delete event objects. |
| ACL | Private. No public list or get. |
| Credentials | Spaces access key available to the sink process only. Not in application runtime for `bidon-sdkapi`. |
| Region | Same Spaces region as existing infra (`ams3` in current Terraform). |

### 4.2 Format and layout

| Item | Requirement |
| --- | --- |
| Encoding | **Parquet** (snappy or zstd). Never JSON, Avro, or CSV as the retained form. |
| Columns | Envelope fields as typed columns + one `payload` JSON column for event-body fields. |
| Layout | Hive: `s3://<bucket>/events/dt=YYYY-MM-DD/event_name=<name>/*.parquet` |
| Partition clock | `dt` from `event_ts` (event wall clock), not ingest wall clock. Late events land on their event day. |
| Join / list predicates | Every path and every query is scoped by `(app_id, …)` (G10). `event_name` partitioning does not replace that. |

**Small files.** At 10k DAU a 10-minute flush is ~41k events / ~8 MiB if written as one file. Partitioning by `event_name` (~40 names) yields many sub-MiB objects — accepted at 10k. Compact, or drop `event_name` from the path, if LIST becomes the bottleneck. Do not shorten the flush to “fix” small files.

Hive day prefixes are the layout. Retention is lifecycle on those prefixes. This TRD does not introduce a table catalog.

### 4.3 Writer

A process we write and operate. Redpanda is the **bus** (Kafka API, already running). It is not the sink.

| Item | Requirement |
| --- | --- |
| Cardinality | **One writer.** Concurrent appends to the same hive prefix corrupt listings. |
| Implementation | **Custom Go binary in this repo** (new `cmd/`, not inside `bidon-sdkapi`). Consume with **franz-go** (already the producer stack). Encode Parquet in-process. PUT objects to Spaces via the S3 API. |
| Not | Redpanda Enterprise Iceberg, Redpanda Connect, Confluent Kafka Connect S3, or any other licensed connector. |
| Input | Topic `telemetry-events` only. Do not consume `ad-events` or `notification-events`. |
| Empty flush | Do not write an object when the batch is empty. |
| Idempotence | At-least-once from Kafka is expected. Duplicate Parquet rows are a query concern (`ROW_NUMBER() OVER (PARTITION BY app_id, event_id)` in the SQL pack), not a writer concern. The sink does not talk to Redis. |

### 4.4 Flush

| Trigger | Value |
| --- | --- |
| Time | **Every 10 minutes** |
| Size | **64 MiB** uncompressed batch, whichever first |
| Empty | Skip |

10-minute lag is acceptable for funnel SQL. Do not page off this lake. Do not flush more often to chase freshness — small files are the cost of that.

### 4.5 Topic (sink input)

The sink cannot start without a topic. Provisioning it is in this TRD because it is a store prerequisite, not because this TRD owns producers.

| Item | Requirement |
| --- | --- |
| Name | `telemetry-events` (env `KAFKA_TELEMETRY_EVENTS_TOPIC`) |
| Creation | **Explicit.** `config/kafka.go` allows auto-create, which yields one partition. |
| Compression | Producer zstd (set at topic / producer; do not rely on the franz-go default). |
| Retention on the bus | **Days**, not 400 d. JSON is ~6× Parquet. The lake is the retain. |
| Partitions | Set from peak JSON/day before first write ([sizing §7](./telemetry-storage-sizing.md#7-tripwires)). |

### 4.6 Query

| Item | Requirement |
| --- | --- |
| Engine | DuckDB `read_parquet('s3://…/events/dt=…/**')`. Not an always-on warehouse. |
| When | On demand. Start DuckDB when someone has a question. |
| Users | Engineers. Not BD / support / finance self-serve. |
| SQL pack | Funnel (request → fill → impression → billing), DSP timeout rate, notice success, `GROUP BY event_name`. Every join on `(app_id, auction_id)`. Counts corrected by `sum(1 / sampling_rate)`. |

### 4.7 Retention

Three drivers, three artefacts. Do not keep one window for everything. Detail: [sizing §4](./telemetry-storage-sizing.md#4-retention).

| Class | Window | Prefix | Contents |
| --- | --- | --- | --- |
| **Aggregates** | **7–10 years** | `aggregates/dt=YYYY-MM-DD/` | Daily rollup: date × app × DSP × ad_format × country → impressions, billable amount, notice successes. No ids, no per-user grain. Statutory **ledger**. |
| Reconciliation | **400 days** | `events/` (lifecycle keep) | `ad_impression`, `billing_notice_sent`, `win_notice_sent`, `auction_completed`, `ad_filled` |
| Diagnostics | **90 days** | `events/` (lifecycle expire) | Everything else, including loss notices |

**Lifecycle.** Spaces/S3 lifecycle rules on `dt=` prefixes. Hive cannot `DELETE` a row; it can drop a day. If legal requires row-level deletes (open item 3), that is a new requirement — not a catalog this TRD ships.

**400 d is an analytics/audit buffer**, not a tax derivation. Confirm the contractual dispute window (AppLovin / DSP agreements) before treating it as settled. Statutory 6–10 year accounting is satisfied by the aggregate, not by replaying auctions.

**Aggregates ship with the sink.** ~10 MB/day, ~25 GiB over seven years. They are computed from never-sampled events, so they do not depend on `K`. They cannot be backfilled once raw rows expire. A scheduled DuckDB (or equivalent) job writes one Parquet file per day to `aggregates/`.

Loss notices sit at 90 d on purpose: only events evidencing a *billing* claim need the reconciliation window.

### 4.8 Failure isolation

| Failure | Ads | Lake |
| --- | --- | --- |
| Sink stalled | Unaffected | Freshness alert; bus retains days of JSON |
| Spaces down / 5xx | Unaffected | Writer backs off; Kafka consumer lag grows |
| DuckDB not running | Unaffected | No one can query until it is started |
| Topic empty (no producers yet) | Unaffected | Lake empty; freshness check must not false-page |

The sink is not on the ad path. Do not add a Spaces or DuckDB dependency to `bidon-sdkapi`.

### 4.9 What the files contain

Schema-on-read. The writer projects known envelope columns and dumps the rest into `payload`. It does **not**:

- Reject a record because `schema_version` is unknown or old. Tag the version, land the row, filter in queries. PRD rationale 6: old SDK majors persist for quarters and never reach zero; dropping them would bias J3/J4.
- Re-sample.
- Strip or rewrite fields (privacy is upstream).
- Interpret `trace_id`. If present it is just another envelope column; this lake does not store or query traces.

---

## 5. Volume and cost

Design point: ~10k DAU, 2 sessions × 5 auctions, fill 0.4, **unsampled** (`K` = 1.0). Model and caveats: [telemetry-storage-sizing.md](./telemetry-storage-sizing.md).

Two unmeasured inputs scale the bill linearly: **1.2 KiB JSON/event on the bus**, **200 B/row in Parquet**. The 200 B figure is optimistic — envelope columns dictionary-encode; the `payload` JSON blob does not. Treat every storage figure as ±2× until the sink produces real bytes.

| | Auctions / day | Events / day | **Parquet / day** | **On Spaces** (90/400 d) | Per 10 min flush | Spaces $/mo |
| --- | --- | --- | --- | --- | --- | --- |
| **10k DAU** | 100k | **5.9M** | **1.1 GiB** | **114 GiB** | ~41k events, ~8 MiB | **~$5** |
| **100k** | 1M | **59M** | **11 GiB** | **1.1 TiB** | ~410k, ~78 MiB | **~$23** |
| **1M** | 10M | **590M** | **110 GiB** | **11 TiB** | ~4.1M, ~780 MiB | **~$230** |

`1.1 GiB/day` is new Parquet. `114 GiB` is retained. JSON on Redpanda is ~6.8 GiB/day at 10k — that is not the Spaces bill.

First measurement after the sink writes: mean Parquet bytes/row, and `event_name` count per `(app_id, auction_id)`. Those two replace the model.

---

## 6. Sink observability

Existing `GET /metrics` on sdkapi is a producer concern and is out of this TRD. The sink needs its own signals:

| Signal | Purpose |
| --- | --- |
| Consumer lag on `telemetry-events` | Bus is backing up |
| Time since last successful Spaces PUT | Writer is stalled |
| Latest object `dt` / modified time vs topic high-water | **Freshness** |
| Objects written / bytes written per flush | Confirms 10 min / 64 MiB triggers |
| Empty-flush skipped | Distinguishes “quiet topic” from “dead writer” |

**Alert:** latest object lags the topic by **> 20 minutes** (two flush intervals). Do not fire that alert when the topic has had no records in the window.

Do not write spans onto `telemetry-events`. Do not scrape this lake for paging.

The PRD client-side loss metric (`telemetry_dropped` ÷ emitted) is an SDK-queue counter. It has no transport through this sink.

---

## 7. Deliverables

| # | Work | Done when |
| --- | --- | --- |
| 1 | Spaces bucket (or dedicated prefix) + private credentials | Terraform or Coolify; not the backups bucket |
| 2 | `telemetry-events` topic created explicitly (partitions, day-scale retention, zstd) | Topic exists before the sink starts |
| 3 | Custom Go Parquet sink (`cmd/…`), 10 min / 64 MiB flush, hive layout as §4.2 | Objects appear under `events/dt=…/event_name=…/`. No Connect, no Enterprise Iceberg. |
| 4 | Lifecycle rules: 90 d diagnostics, 400 d reconciliation, 7–10 y aggregates | Day prefixes expire; aggregates do not |
| 5 | Daily aggregate job writing `aggregates/dt=…/` | One Parquet file per day; cannot wait until raw retention is a problem |
| 6 | DuckDB SQL pack (funnel, DSP timeout, notice success, `GROUP BY event_name`) | An engineer can answer J3/J4 against a day of files |
| 7 | Freshness alert (> 20 min, quiet-topic excluded) | Page on stall, not on idle |

Local/dev: MinIO or a Spaces staging bucket is enough. Do not require production Spaces to prove the writer.

---

## 8. Acceptance

This TRD is done when all of the following are true:

1. A JSON record on `telemetry-events` becomes a Parquet row under `events/dt=<event day>/event_name=<name>/` within **10 minutes** under load below the 64 MiB trigger.
2. DuckDB `read_parquet` over a day partition returns that row. Funnel SQL in the pack runs, scoped by `app_id`, correcting with `sampling_rate`.
3. No object is JSON. No writer reads `ad-events`. The writer is the custom Go binary in this repo — not Connect, not Redpanda Iceberg.
4. `bidon-sdkapi` has no Spaces or DuckDB import; killing the sink does not change auction or `/show` behaviour.
5. A day of diagnostics older than 90 d is gone; reconciliation event names survive to 400 d; the aggregate prefix is excluded from those expiries.
6. After one scheduled run, `aggregates/dt=<day>/` contains a rollup file.
7. Freshness pages when the writer is stalled with records on the topic, and does not page when the topic is idle.

It is **not** a failure of this TRD that the topic is empty, that `/v2/telemetry` does not exist, or that traces and metrics are unset.

---

## 9. Open items

Nothing below is architecture-blocked. Every item needs a **name**. Items that belong to ingest, traces, or metrics are not listed.

| # | Item | Blocks | Owner |
| --- | --- | --- | --- |
| 1 | Who provisions the Spaces bucket and runs the custom sink | This TRD entirely | **unnamed** |
| 2 | Contractual dispute / audit window (MAX terms, DSP agreements) — is 400 d right? | Reconciliation lifecycle | Commercial / counsel |
| 3 | Legal: is `session_id` + `device_model` + `app_bundle` + `country` pseudonymous PII requiring DSRs? | Whether day-prefix lifecycle is enough | Counsel |
| 4 | Measured Parquet bytes/row and `event_name` counts per auction | Turns §5 from a model into a bill | Backend, once the sink writes |

---

## 10. PRD traceability (this store only)

| PRD requirement | This TRD | Status |
| --- | --- | --- |
| Raw events to a columnar warehouse for funnel joins | Entire document | Parquet on Spaces, DuckDB on demand |
| Funnel completeness, fill / render / notice rates (catalog SQL) | §4.6 SQL pack | Engineer-mediated in this slice |
| Goal 3 reconciliation (impression-level rows) | §4.7 400 d class + aggregates | Window unconfirmed commercially |
| Retention per event class before launch | §4.7 | Lifecycle + aggregate job |
| Schema_version additive; do not break downstream queries | §4.9 land every version | Writer does not reject |
| Telemetry must not delay the ad path | §4.8 | Sink is off the ad path |
| Sampling rate recorded so counts correct at query time | §4.6 `sum(1 / sampling_rate)` | Rate is an upstream field; lake only reads it |
| OTel one trace / DSP span | — | **Out** |
| Metrics for alerting | — | **Out** |
| Client disk queue, ingest, kill switch | — | **Out** |
| No PII on the stream | Assumed upstream | Writer does not enforce |

Any PRD infrastructure dependency not listed here is out of this TRD, not an implicit decision.
