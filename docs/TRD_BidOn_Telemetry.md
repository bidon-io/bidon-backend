# Technical Requirements Document — Telemetry event storage

**Parquet on DigitalOcean Spaces**

**Status:** Draft  
**Date:** 2026-09-01  
**This TRD specifies:** the events warehouse. One row per product event, retained as Parquet on DigitalOcean Spaces, flushed every 10 minutes.

| | |
| --- | --- |
| Why the warehouse exists | [PRD v1](./PRD_BidOn_Telemetry%20-%20v1.pdf) — Goals 1–4; Part II “raw events to a columnar warehouse for funnel joins” |
| What the warehouse must answer | [telemetry-requirements.md](./telemetry-requirements.md) §2.A (jobs J3, J4, J5) |
| Why Parquet on object storage | [telemetry-events-store.md](./telemetry-events-store.md) |
| Numbers | [telemetry-storage-sizing.md](./telemetry-storage-sizing.md) |

No production instrumentation ships from this draft. The lake can exist empty; producers are a different piece of work.

---

## 1. Scope

- A DigitalOcean Spaces bucket (S3 API) for telemetry events.
- **Protobuf** schemas, one message type per event, in Schema Registry.
- A **Kafka Connect** S3 sink (Confluent Community License — not a paid SKU) that reads those protobuf records and writes **Parquet**.
- **One table per event type** (own prefix, own schema). Day partitions inside each table.
- 10-minute flush, compression.
- Lifecycle retention **per table** (400 d reconciliation tables, 90 d diagnostics), plus a daily aggregate table.
- A DuckDB SQL pack an engineer can run against the bucket.
- A freshness alert on the sink.

The sink reads Confluent-wire **protobuf** already on `telemetry-events` (magic byte + schema id + message). Each record is a registered message type. `sampling_rate` is a field on the message; the sink does not re-sample and does not drop by rate.

If the topic is empty, the lake is empty. That is acceptable for this TRD.

---

## 2. Why this store

The PRD’s jobs that need a warehouse are catalog SQL:

| Job | Query the lake must support |
| --- | --- |
| J3 | Fill rate, time-to-fill, render rate, per-source load — `sum(1 / sampling_rate)`, `GROUP BY` demand |
| J4 | Reconcile Bidon impressions vs mediator / DSP billing |
| J5 | Per-DSP loss and render |

Requirements §2.A: if a system cannot do those joins, it is not the event store. Minutes of lag are fine.

The lake is downstream of Redpanda. If the sink or Spaces is down, ads still serve (G1/G2).

**Who runs the SQL.** For this slice those jobs are engineer-mediated: someone runs the saved SQL pack and returns numbers.

---

## 3. Pipeline

```
Redpanda telemetry-events (protobuf + Schema Registry)
        │
        ▼
Kafka Connect S3 sink  ── ParquetFormat, flush every 10 min
        │
        ▼
DigitalOcean Spaces
  events/<event_name>/dt=YYYY-MM-DD/*.parquet   ← one table per event type
  aggregates/dt=YYYY-MM-DD/*.parquet            ← daily rollup (its own table)
        │
        ▼
DuckDB  (read_parquet per table, join on envelope keys)
```

Protobuf on the topic is the **bus**. Parquet on Spaces is the **store**.

The existing Connect connector (`docker/kafka-connect`, JSON of `ad-events`) stays as that archive. This warehouse is a **new** connector on `telemetry-events`.

---

## 4. Requirements

### 4.1 Bucket

| Item | Requirement |
| --- | --- |
| Service | DigitalOcean **Spaces** (S3 API) |
| Isolation | Dedicated telemetry bucket, or a dedicated prefix on a bucket that is **not** the Postgres-backup bucket (`infra` currently creates `${name_prefix}-backups-*`). Backup lifecycle must not delete event objects. |
| ACL | Private. No public list or get. |
| Credentials | Spaces access key on the **Connect worker** only. Not in `bidon-sdkapi`. |
| Region | Same Spaces region as existing infra (`ams3` in current Terraform). |

### 4.2 Protobuf schemas

Strict, per event type. The Parquet columns **are** the protobuf fields.

| Item | Requirement |
| --- | --- |
| Source | `.proto` files in this repo (alongside existing `proto/`). Generated Go for producers; the registry is the runtime source for Connect. |
| Shape | One message type per PRD `event_name` (`AuctionCompleted`, `DspRequestSent`, …). Shared envelope fields live in a reusable message **embedded** in every event (not a JSON blob). |
| Wire | Confluent protobuf: magic byte, schema id, protobuf payload. |
| Registry | Schema Registry (already in `docker-compose-prod.yml`). Subject strategy **TopicRecordNameStrategy** so each message type has its own subject on `telemetry-events`. |
| Compatibility | **BACKWARD** per subject. Additive field numbers only; never reuse a number. Matches PRD “additive within a major.” |
| Versions | The sink lands every registered schema id. It does not reject old majors. DuckDB reads mixed file versions with `union_by_name=true`. PRD rationale 6: old SDK (and old producer) schemas persist for quarters. |

Envelope fields required on every message (embedded): `event_name`, `event_id`, `event_ts`, `schema_version`, `app_id`, `auction_id`, `session_id`, `sampling_rate`, plus the rest of the PRD common envelope as protobuf fields.

Type-specific fields are first-class columns (`dsp_id`, `http_status`, `clearing_price`, …), not packed into `payload`.

### 4.3 Tables, format, and layout

A table has one schema. Strict protobuf per event type therefore means **one table per `event_name`**, not one `events` table partitioned by type. Hive `event_name=` as a partition of a unified table would imply shared columns; these messages do not share columns beyond the embedded envelope.

| Item | Requirement |
| --- | --- |
| Table | One per PRD event type. Name = `event_name` (`auction_completed`, `dsp_request_sent`, …). ~40 tables at full catalog. |
| Encoding | **Parquet** (`ParquetFormat`, snappy or zstd). |
| Columns | That type’s protobuf fields (envelope + type-specific). |
| Layout | `s3://<bucket>/events/<event_name>/dt=YYYY-MM-DD/*.parquet` |
| Partition clock | `dt` from `event_ts` (event wall clock), not ingest wall clock. Late events land on their event day. |
| Homogeneous files | A file belongs to one table. Connect Parquet cannot hold two schemas. |
| Joins | Envelope keys `(app_id, auction_id)` (G10). Funnel SQL is **JOIN across tables**, not `GROUP BY event_name` on a union. |
| Schema evolution | `union_by_name` **within** a table (old vs new field numbers of the same message). Do not union different event types into one scan. |

**Connect partitioner.** `FieldPartitioner` on `event_name` (table directory) plus a `dt` field materialised from `event_ts` (SMT `TimestampConverter` or equivalent). One connector writes all tables.

**Small files.** At 10k DAU a 10-minute flush splits across ~40 tables. Accepted at 10k. Compact **inside** a table if LIST hurts. Do not merge types into one prefix.

**Not a table:** a fat protobuf `oneof` wrapping every event, or one Parquet schema with every field nullable. That undoes strict per-type schemas.

### 4.4 Writer

Kafka Connect S3 sink, Confluent Community License (no fee; competing-SaaS restriction only). Redpanda is the bus.

| Item | Requirement |
| --- | --- |
| Cardinality | **One task** (`tasks.max=1`). |
| Connector | New: `telemetry-events-parquet`. Class `io.confluent.connect.s3.S3SinkConnector`. |
| Format | `io.confluent.connect.s3.format.parquet.ParquetFormat` |
| Value converter | `io.confluent.connect.protobuf.ProtobufConverter` + Schema Registry URL. Not `JsonConverter` (NPE with ParquetFormat). |
| Input | Topic `telemetry-events` only. |
| Spaces | Path-style, `store.url` for the Spaces endpoint, keys from Connect env. |
| Empty flush | Do not write an object when the batch is empty. |
| Idempotence | At-least-once. Duplicate Parquet rows are a query concern (`ROW_NUMBER() OVER (PARTITION BY app_id, event_id)`). |

Reuse the existing Connect **worker** (`docker/kafka-connect`). Do not retarget the JSON `ad-events` connector.

### 4.5 Flush

| Trigger | Value |
| --- | --- |
| Time | **Every 10 minutes** — `rotate.schedule.interval.ms=600000` (set `timezone=UTC`) |
| Size | `flush.size` high enough that **time wins at 10k DAU** (~41k events / 10 min). Revisit once protobuf bytes/row are measured. |
| Empty | Skip |

10-minute lag is acceptable for funnel SQL. Do not flush more often to chase freshness — small files are the cost of that.

### 4.6 Topic (sink input)

The sink cannot start without a topic. Provisioning it is in this TRD because it is a store prerequisite.

| Item | Requirement |
| --- | --- |
| Name | `telemetry-events` (env `KAFKA_TELEMETRY_EVENTS_TOPIC`) |
| Creation | **Explicit.** `config/kafka.go` allows auto-create, which yields one partition. |
| Compression | Producer zstd (set at topic / producer; do not rely on the franz-go default). |
| Retention on the bus | **Days**, not 400 d. The lake is the retain. |
| Partitions | Set from peak protobuf/day before first write ([sizing §7](./telemetry-storage-sizing.md#7-tripwires)). |

### 4.7 Query

| Item | Requirement |
| --- | --- |
| Engine | DuckDB. Scan **one table** at a time: `read_parquet('s3://…/events/ad_impression/dt=…/**', union_by_name=true)`. Not an always-on warehouse. |
| When | On demand. Start DuckDB when someone has a question. |
| Users | Engineers. |
| SQL pack | Funnel is joins: `ad_request_started` ⋈ `ad_filled` ⋈ `ad_impression` ⋈ `billing_notice_sent` on `(app_id, auction_id)`, each from its own table. DSP timeout and notice success similarly. Counts corrected by `sum(1 / sampling_rate)`. |

### 4.8 Retention

Three drivers, three artefacts. Do not keep one window for everything. Detail: [sizing §4](./telemetry-storage-sizing.md#4-retention).

| Class | Window | Where | Contents |
| --- | --- | --- | --- |
| **Aggregates** | **7–10 years** | table `aggregates/` | Daily rollup: date × app × DSP × ad_format × country → impressions, billable amount, notice successes. No ids, no per-user grain. Statutory **ledger**. |
| Reconciliation | **400 days** | those **tables** only | `ad_impression`, `billing_notice_sent`, `win_notice_sent`, `auction_completed`, `ad_filled` |
| Diagnostics | **90 days** | every other event table | Including loss notices |

**Lifecycle is per table prefix**, not a row filter inside a mixed `events/` folder. Spaces rules on `events/ad_impression/` vs `events/loss_notice_sent/` (and the other names in each class). Hive cannot `DELETE` a row; it can drop a day of a table.

**400 d is an analytics/audit buffer**, not a tax derivation. Confirm the contractual dispute window (AppLovin / DSP agreements) before treating it as settled. Statutory 6–10 year accounting is satisfied by the aggregate, not by replaying auctions.

**Aggregates ship with the sink.** ~10 MB/day, ~25 GiB over seven years. They are computed from never-sampled events, so they do not depend on `K`. They cannot be backfilled once raw rows expire. A scheduled DuckDB (or equivalent) job writes one Parquet file per day to `aggregates/`.

Loss notices sit at 90 d on purpose: only events evidencing a *billing* claim need the reconciliation window.

### 4.9 Failure isolation

| Failure | Ads | Lake |
| --- | --- | --- |
| Connect stalled | Unaffected | Freshness alert; bus retains days of protobuf |
| Schema Registry down | Unaffected (produce may fail — producer concern) | Sink cannot decode new ids until SR returns |
| Spaces down / 5xx | Unaffected | Connector backs off; Kafka consumer lag grows |
| DuckDB not running | Unaffected | No one can query until it is started |
| Topic empty (no producers yet) | Unaffected | Lake empty; freshness check must not false-page |

The sink is not on the ad path. Do not add a Spaces, Connect, or DuckDB dependency to `bidon-sdkapi`.

---

## 5. Volume and cost

Design point: ~10k DAU, 2 sessions × 5 auctions, fill 0.4, **unsampled** (`K` = 1.0). Model and caveats: [telemetry-storage-sizing.md](./telemetry-storage-sizing.md).

Sizing still uses **1.2 KiB/event on the bus** and **200 B/row Parquet** as unmeasured placeholders (those were JSON-era). Protobuf + typed columns should compress better than a `payload` JSON blob; treat figures as ±2× until the sink produces real bytes.

| | Auctions / day | Events / day | **Parquet / day** | **On Spaces** (90/400 d) | Per 10 min flush | Spaces $/mo |
| --- | --- | --- | --- | --- | --- | --- |
| **10k DAU** | 100k | **5.9M** | **1.1 GiB** | **114 GiB** | ~41k events | **~$5** |
| **100k** | 1M | **59M** | **11 GiB** | **1.1 TiB** | ~410k | **~$23** |
| **1M** | 10M | **590M** | **110 GiB** | **11 TiB** | ~4.1M | **~$230** |

`1.1 GiB/day` is new Parquet. `114 GiB` is retained. Bus retention is days, not 400 d.

First measurement after the sink writes: mean protobuf bytes on the wire, Parquet bytes/row, and `event_name` count per `(app_id, auction_id)`.

---

## 6. Sink observability

| Signal | Purpose |
| --- | --- |
| Consumer lag on `telemetry-events` | Bus is backing up |
| Connect task state / failed records | Decoder or PUT errors |
| Time since last successful Spaces PUT | Writer is stalled |
| Latest object `dt` / modified time vs topic high-water | **Freshness** |
| Objects written / bytes written per flush | Confirms the 10 min trigger |

**Alert:** latest object lags the topic by **> 20 minutes** (two flush intervals). Do not fire that alert when the topic has had no records in the window.

---

## 7. Deliverables

| # | Work | Done when |
| --- | --- | --- |
| 1 | Spaces bucket (or dedicated prefix) + private credentials | Terraform or Coolify; not the backups bucket |
| 2 | Schema Registry reachable from Connect (prod compose already has it; dev/Coolify must too) | Protobuf converter can fetch ids |
| 3 | Protobuf event messages in-repo + subjects registered (BACKWARD, TopicRecordNameStrategy) | At least one event type round-trips |
| 4 | `telemetry-events` topic created explicitly (partitions, day-scale retention, zstd) | Topic exists before the sink starts |
| 5 | New Connect connector `telemetry-events-parquet`: ProtobufConverter, ParquetFormat, 10 min rotate, one table prefix per `event_name` + `dt=`, Spaces endpoint | Objects appear under `events/<event_name>/dt=…/` as Parquet |
| 6 | Lifecycle rules **per table**: 90 d diagnostics tables, 400 d reconciliation tables, 7–10 y aggregates | Day prefixes expire; the two classes differ by table |
| 7 | Daily aggregate job writing the `aggregates` table | One Parquet file per day |
| 8 | DuckDB SQL pack (per-table scans, funnel as joins) | An engineer can answer J3/J4 against a day of files |
| 9 | Freshness alert (> 20 min, quiet-topic excluded) | Page on stall, not on idle |

Local/dev: MinIO or a Spaces staging bucket is enough. Do not require production Spaces to prove the writer.

---

## 8. Acceptance

This TRD is done when all of the following are true:

1. A protobuf record on `telemetry-events` (registered schema id) becomes a Parquet row under `events/<event_name>/dt=<event day>/` within **10 minutes** under load where time-based rotate fires first.
2. That file’s columns match that event’s protobuf message. DuckDB `read_parquet` on **that table** (with `union_by_name` for schema versions of the same type) returns the row. Funnel SQL joins tables on `(app_id, auction_id)` and corrects with `sampling_rate`.
3. Objects are Parquet. The writer is the new Connect connector, consuming `telemetry-events` only. Each `events/<event_name>/` prefix is one table: one message type, one schema.
4. `bidon-sdkapi` has no Spaces, Connect, or DuckDB import; killing Connect does not change auction or `/show` behaviour.
5. Diagnostics tables older than 90 d are gone; reconciliation tables survive to 400 d; `aggregates/` is excluded from those expiries.
6. After one scheduled run, the `aggregates` table contains a rollup file for that day.
7. Freshness pages when the writer is stalled with records on the topic, and does not page when the topic is idle.

---

## 9. Open items

Nothing below is architecture-blocked. Every item needs a **name**.

| # | Item | Blocks | Owner |
| --- | --- | --- | --- |
| 1 | Who provisions Spaces, Schema Registry (dev/Coolify), and the Connect connector | This TRD entirely | **unnamed** |
| 2 | Contractual dispute / audit window (MAX terms, DSP agreements) — is 400 d right? | Reconciliation lifecycle | Commercial / counsel |
| 3 | Measured protobuf bytes/event and Parquet bytes/row | Turns §5 from a model into a bill | Backend, once the sink writes |

---

## 10. PRD traceability

| PRD requirement | This TRD | Status |
| --- | --- | --- |
| Raw events to a columnar warehouse for funnel joins | Entire document | Protobuf → Connect → Parquet on Spaces, DuckDB on demand |
| Funnel completeness, fill / render / notice rates (catalog SQL) | §4.7 SQL pack | Engineer-mediated in this slice |
| Goal 3 reconciliation (impression-level rows) | §4.8 400 d class + aggregates | Window unconfirmed commercially |
| Retention per event class before launch | §4.8 | Lifecycle + aggregate job |
| Schema versioned; additive within a major | §4.2 BACKWARD, additive field numbers | Sink lands every registered id |
| Telemetry must not delay the ad path | §4.9 | Sink is off the ad path |
| Sampling rate recorded so counts correct at query time | §4.7 `sum(1 / sampling_rate)` | Field on the protobuf message |
