# Product events: Parquet on S3, SQL when needed, Iceberg when ready

**Scope:** events only ([telemetry-requirements.md](./telemetry-requirements.md) §2.A). Traces: [telemetry-traces-store.md](./telemetry-traces-store.md). Metrics: [telemetry-metrics-store.md](./telemetry-metrics-store.md).  
**Phase 1 implementation contract:** [TRD_BidOn_Telemetry.md](./TRD_BidOn_Telemetry.md) — Parquet on DigitalOcean Spaces, 10 min flush. This file is the design and later phases.  
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
                         → custom Go sink (flush every **10 min**, or 64 MiB, whichever first)
                         → DigitalOcean Spaces: s3://…/events/dt=YYYY-MM-DD/event_name=…/*.parquet
                         → DuckDB: read_parquet('s3://…/events/dt=…/**')
```

- Envelope columns + `payload` JSON. Dual-emit old topics unchanged.  
- Redis `event_id` at ingest, client events only. Freshness = latest object vs topic; alert if lag **> 20 min**.  
- Saved SQL: funnel, DSP timeout, notice success, `GROUP BY event_name` — scoped by `(app_id, auction_id)` (G10).

**Good enough** to size sampling and prove the catalog. Hive paths are a **temporary** layout.

At the first-iteration design point (5 auctions/session, unsampled): **1.1 GiB/day Parquet, ~114 GiB retained on Spaces** ([sizing](./telemetry-storage-sizing.md), [TRD §5](./TRD_BidOn_Telemetry.md#5-volume-and-cost)). JSON on the bus is ~6.8 GiB/day at 10k DAU; that is not the Spaces bill.

The `payload` JSON blob column is why the 200 B/row estimate is optimistic — envelope columns dictionary-encode well, an opaque JSON string does not. Measure bytes/row in Phase 1 and feed it back into the sizing model before quoting a bill.

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

- Lake lag (**10 min** flush). Fine for PRD analytics; not for paging.  
- Small files if you flush too often.  
- **No SQL `DELETE` in Phase 1.** Retention is lifecycle rules on day partitions, which works. Row-level deletion does not. See the GDPR note below — this is the one drawback that could reorder the phases.
- **DuckDB ≠ BI farm, and J3/J4/J5 name non-engineers.** BD, publisher support, solutions and finance do not run DuckDB over object storage. For M0/M1 those jobs are engineer-mediated: someone runs the saved SQL pack and returns numbers. That is a deliberate scope choice, but it needs saying out loud to those teams, because "we have telemetry now" will be heard as "I can self-serve." A query surface for non-engineers is M2.
- Sink can stall while old Kafka looks healthy — alert freshness.  
- Iceberg needs a **catalog** process; hive Parquet does not. That is why it is Phase 2, not a blocker for generating events.

### The GDPR question that could promote Phase 2

If G3 holds strictly there is no personal data on this stream and nothing to delete — retention by day-partition lifecycle is sufficient and hive Parquet is fine.

But `session_id` + `device_model` + `app_bundle` + `country` is plausibly **pseudonymous** personal data under GDPR. If legal says data-subject requests must be honoured against this lake, row-level delete becomes a requirement: hive-partitioned Parquet cannot do it, Iceberg can.

**Ask legal before committing to the Phase 1 layout.** A "yes" moves Iceberg from Phase 2 to M0 and changes what the sink has to be on day one. A "no" costs one conversation.

---

## Generate / consume

| Phase | Writer | What lands in the bucket |
| --- | --- | --- |
| 1 | Custom Go consumer in this repo (franz-go → Parquet → Spaces). Not Redpanda Enterprise, not Connect. | Hive-style Parquet |
| 2+ | OSS Iceberg writer if a catalog is ever added — still not Redpanda Enterprise | Parquet **plus** Iceberg metadata + catalog pointer |

JSON on the topic never changes.

## M0 slice

1. `telemetry-events` + custom Go Phase 1 sink ([TRD](./TRD_BidOn_Telemetry.md)).  
2. DuckDB SQL pack.  
3. **Daily aggregate rollup** written to a separate long-retention prefix ([sizing §4](./telemetry-storage-sizing.md#4-retention)) — a scheduled query, ~25 GiB over seven years, and the only artefact that survives a decision to shorten raw retention. It cannot be backfilled once raw rows expire, so it ships with the sink.  
4. Freshness alert on the sink.  
5. Written tripwire for Phase 4 (CH).
