# Product events: Parquet on S3, SQL when needed, Iceberg when ready

**Scope:** grain A only ([telemetry-requirements.md](./telemetry-requirements.md) §2.A). Grain B: [telemetry-traces-store.md](./telemetry-traces-store.md) (VictoriaTraces from day 1). Grain C: [telemetry-metrics-store.md](./telemetry-metrics-store.md) (VictoriaMetrics from day 1).  
**Decision:** JSON → OSS Redpanda → **Parquet on object storage** → **DuckDB** (or Athena/Trino) when you query. **Iceberg** is the table format to adopt when an OSS writer + REST catalog are in place — not Redpanda Enterprise, not Connect’s licensed `iceberg` output. **ClickHouse** later reads the **same** lake.  
**Sizing:** [telemetry-storage-sizing.md](./telemetry-storage-sizing.md).

These are **phased**. Each phase keeps the wire format and `/v2/show` → Redpanda produce path.

`/v2/show` produces events (and fires BURL). It does not query the lake.

---

## Argument

Catalog SQL is delivery evidence (`sum(1 / sampling_rate)`, `GROUP BY dsp_id`, join `auction_id`), typically minutes to T+1 — normal in ad analytics. Real-time is BURL and later metrics/traces, not DuckDB.

Redpanda is already paid for. Incremental store = sink that writes **Parquet**, plus an engine you start **when you have a question**.

**Parquet** is the file encoding (never JSON-on-S3 for this catalog). **Iceberg** is a catalog + snapshots *over* those files so engines do not glob `dt=*`. **DuckDB** is the first engine (CLI/CI, `read_parquet` then Iceberg extension). CH is a query accelerator on the same files when load requires it.

---

## Phases

### Phase 1 — Lake exists (M0 minimum)

```
bidon-sdkapi (JSON) → Redpanda telemetry-events
                         → OSS consumer (Connect S3 parquet *or* a small writer)
                         → s3://…/dt=YYYY-MM-DD/event_name=…/*.parquet
                         → DuckDB: read_parquet('s3://…/dt=…/**')
```

- Envelope columns + `payload` JSON. Large parts (tens of MB). Dual-emit old topics unchanged.  
- Redis `event_id` at ingest. Freshness = latest object vs topic lag.  
- Saved SQL: funnel, DSP timeout, notice success, `GROUP BY event_name`.

**Good enough** to size sampling and prove the catalog. Hive paths are a **temporary** layout.

### Phase 2 — Iceberg table (OSS, no Enterprise)

Do **not** wait for `redpanda.iceberg.mode` or Connect `output: iceberg:` (both license-gated).

```
same topic → PyIceberg (or Iceberg Java) consumer
                → Parquet data files
                → snapshot/manifest commit
                → REST catalog (Polaris or iceberg-rest + MinIO/R2/S3)
                → DuckDB Iceberg extension
```

`table.append(batch)` **is** writing Iceberg metadata. One writer, one table `bidon.telemetry_events`, partition on day(`event_ts`) + `event_name`.

If Phase 1 already has hive Parquet: convert **before** file counts explode (rewrite into Iceberg once), or skip hive and start Phase 2 as soon as the catalog runs in Compose.

### Phase 3 — Query load (optional engines)

Same files/catalog. Add Athena (metered scan) or Trino only if DuckDB-over-S3 is the bottleneck. Not a new schema.

### Phase 4 — ClickHouse (tripwire)

When queries are daily/hourly, or DuckDB misses “minutes, not an afternoon,” or many concurrent SQL users: CH `Iceberg`/`S3` table on **that catalog**. No second event model. Do not buy CH Cloud to skip Phase 1–2.

---

## Drawbacks (accepted)

- Lake lag (flush interval). Fine for PRD analytics; not for paging.  
- Small files if you flush too often.  
- No SQL `DELETE`; lifecycle/rewrite for retention and GDPR.  
- DuckDB ≠ BI farm.  
- Sink can stall while old Kafka looks healthy — alert freshness.  
- Iceberg needs a **catalog** process; hive Parquet does not. That is why it is Phase 2, not a blocker for generating events.

---

## Generate / consume

| Phase | Writer | What lands in the bucket |
| --- | --- | --- |
| 1 | Connect community S3 parquet, or any consumer | Hive-style Parquet |
| 2+ | PyIceberg `append` (OSS) | Parquet **plus** Iceberg metadata + catalog pointer |

JSON on the topic never changes.

## M0 slice

1. `telemetry-events` + Phase 1 sink (or Phase 2 if catalog is ready in the same sprint).  
2. DuckDB SQL pack.  
3. Freshness alert on the sink.  
4. Written tripwire for Phase 4 (CH).
