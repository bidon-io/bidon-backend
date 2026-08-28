# Technical Requirements Document — BidOn Telemetry (backend)

**Status:** Draft  
**Date:** 2026-08-28  
**Start here:** [telemetry-brief.md](./telemetry-brief.md)  
**Companion PRD:** [PRD_BidOn_Telemetry - v1.pdf](./PRD_BidOn_Telemetry%20-%20v1.pdf)  
**Backend spike / mapping:** [telemetry-m0-m1-backend-spike.md](./telemetry-m0-m1-backend-spike.md)  
**Requirements:** [telemetry-requirements.md](./telemetry-requirements.md)  
**Settled for this TRD:** grain A only — [telemetry-events-store.md](./telemetry-events-store.md): JSON → Redpanda `telemetry-events` → **Parquet on DigitalOcean Spaces**, **10 min flush**.  
**Out of this TRD:** traces (B) and metrics (C). See [telemetry-traces-store.md](./telemetry-traces-store.md) / [telemetry-metrics-store.md](./telemetry-metrics-store.md) when those are implemented.  
**Sizing:** [telemetry-storage-sizing.md](./telemetry-storage-sizing.md)

The PRD states *what* must be observable. This TRD states *what carries it* on the Bidon server. Client transport (on-disk queue, crash path, retry-to-ingest) is SDK-owned (BID-47); this document only specifies the ingest contract that queue talks to, and the server-native event path.

No production instrumentation ships from this draft.

---

## 1. Design constraints

- Telemetry must not delay or fail an ad request. Server-native emits are fire-and-forget, same as today's `event.Logger.Log`.
- Exception: `ad_impression` / billing stay on the **ad-serving channel** (`POST /v2/show/{ad_type}`), not the SDK batch queue.
- New stream has **no PII**. The existing `ad-events` topic is unchanged during dual-emission and still contains PII; it is not the telemetry product.
- Additive schema changes within a `schema_version` major; bump major for breaking changes.
- Errors are never sampled. Sampling is **coherent per auction** (requirements G11), never independent per event.
- Every join, dedupe key and partition predicate is scoped by `(app_id, …)` (requirements G10). `auction_id` is client-supplied and not globally unique.

**Scale posture.** First iteration of **events only**. Design point ~10k DAU, headroom to 1M. Traces and metrics are not in this implementation.

---

## 2. Client transport (ingest-facing contract)

SDK implements the queue. Server requires:

| Rule | Value |
| --- | --- |
| Endpoint | `POST /v2/telemetry` on `bidon-sdkapi` |
| Auth | Same as other SDK routes: `app.key` + `app.bundle` lookup; `X-Bidon-Version` |
| Content-Type | `application/json` body (not query params) |
| Batch | Array of envelope objects, max **100 events** or **256 KiB** uncompressed, whichever first. Oversize batch → `413` |
| Compression | `Content-Encoding: gzip` accepted |
| Clock | Client `event_ts` is wall clock ms; server also stores `received_ts` |
| Sampling | Client stamps `sampling_rate` from `/v2/config`. Server does **not** re-sample. Missing rate on a sampled event class → treat as `1.0` and increment a metric |
| Kill switch | If config says publisher kill switch on, SDK should not send. If it does, ingest accepts and drops with `telemetry_dropped` reason `kill_switch` (still 200 so the SDK queue drains) |
| Retry | SDK retries network and 5xx, up to 5, TTL-bounded (BID-47). **4xx is not retried.** Ingest must not return 4xx for transient overload |
| Idempotency | Dedupe on `event_id` (UUID). Duplicate within retention → 200, not stored twice |

Success response: `200 { "accepted": N, "duplicate": M, "rejected": K }`. Per-event reject reasons in a bounded array (cap 20) so a poison event is visible without huge payloads.

This endpoint is new. Domain endpoints (`/auction`, `/stats`, `/show`, …) keep their current contracts.

---

## 3. The `ad_impression` exception

### Mechanism

`POST /v2/show/{ad_type}` is the delivery-guaranteed path.

1. Handler authenticates and validates as today.
2. For bidding ads, `HandleShow` fires BURL (billing notice) with existing retries.
3. Handler emits `ad_impression` and, after the BURL attempt, `billing_notice_sent` or `notice_delivery_failed` onto `telemetry-events`.
4. HTTP 200 to the SDK means Bidon accepted the impression for billing purposes, **independent of Kafka**.

Kafka remaining down after `/show` 200 is a **pipeline** loss, not a billing loss. Alert on produce failures; do not fail `/show`.

### If `/show` is unavailable

No impression, no BURL. That is existing product behaviour. Telemetry does not invent a second billing trigger.

### SDK also sending `ad_impression`

Allowed for renderer-side diagnostics. Dedupe:

- Prefer client `event_id` if `/show` echoes or accepts an `event_id`.
- Transition key if not: `show:{auction_id}:{demand_id}:{ad_unit_uid}`.

**Source of truth for `billing_notice_sent`:** the server BURL attempt from `/show`, never the SDK batch event.

### Degradation

| Failure | Billing | Telemetry |
| --- | --- | --- |
| `/show` 4xx/5xx | No BURL | No server `ad_impression` |
| BURL transport failure after retries | Unbilled win (existing, detected) | `notice_delivery_failed` |
| BURL returns 4xx/5xx | Unbilled win | **Today: undetected.** See below |
| Kafka produce fail | BURL already sent | Missing warehouse row; `telemetry_produce_failed` metric |

### Known defect: notice outcomes are not observed today

`internal/notification/event_sender.go:52-60` returns `nil` from the retry closure for **any** HTTP response. Only transport errors are caught; a 500 from a DSP's BURL endpoint is recorded as delivered.

Consequence: **J4 (reconcile impressions against mediator/DSP billing) and PRD Goal 3 are not achievable at all until BE-M1-3 lands.** This makes BE-M1-3 the highest-value ticket for the PRD's stated business case, not a late M1 item — reflected in the sequencing in the [spike](./telemetry-m0-m1-backend-spike.md#suggested-sequence).

Related, and separate from telemetry: `internal/notification/handler.go:234` selects the bid to bill with `bid.Price == impression.GetPrice()` — float equality across a client round-trip, first-match-wins when two DSPs bid identically. For an initiative whose Goal 3 is reconciliation within an agreed tolerance, the billing notice can be attributed to the wrong demand. Instrumenting this path will surface it; better to fix it deliberately than to discover it during certification.

---

## 4. Ingest

### 4.1 Server-native producer

Package `internal/telemetry` (new), used by:

- `internal/auction/service.go` — `auction_request_received`, `auction_completed`
- `internal/bidding/builder.go` — `dsp_request_sent` (before `ExecuteRequest`), `dsp_response_received` / `dsp_response_rejected` (after `ParseBids`)
- `internal/notification/event_sender.go` — notice events with HTTP outcome

Same `LoggerEngine` pattern as `internal/sdkapi/event`, writing `telemetry-events`.

`dsp_request_sent` must be timestamped at **send**, not auction start. Today's `DemandResponse.StartTS` is `params.StartTS` (auction start) and is the wrong clock for this event.

### 4.2 Authentication and app identity

Reuse `AppFetcher.FetchCached`. Unknown app → `422` (consistent with other SDK routes, not 401, unless product changes all routes together).

`publisher_id` on the envelope is the Bidon app/user owner id from the fetched app, not a client-supplied field.

### 4.3 `event_id` dedupe

- Client events: client-generated UUID. **Dedupe applies to these only.**
- Server events: server-generated UUID at emit time. A UUID minted in-process cannot arrive twice — do **not** spend a Redis entry on it. ~15 of the ~55 raw events per auction are server-minted.
- Store: Redis SET with TTL **72h**, key `tel:evt:{app_id}:{event_id}` (scoped per G10). Cluster already required (`REDIS_CLUSTER`).
- **Redis-down policy: fail open.** Accept the batch and skip the dedupe check. G1 outranks G5, and read-time dedupe catches whatever slipped through.
- **Tripwire.** Redis dedupe is ~2.5 GiB at 10k DAU (raw client events, 72 h). At 1M DAU, still unsampled, it is **~250 GiB of RAM**. At that point move to read-time dedupe (`ROW_NUMBER() OVER (PARTITION BY event_id)`); the lake is append-only and every query is batch, so G5 holds logically without RAM. Recorded in [sizing §7](./telemetry-storage-sizing.md#7-tripwires).
- **TTL is a joint contract with BID-47.** The SDK queue is disk-backed and survives process death. If its TTL exceeds 72h, dedupe silently fails for exactly the crash-recovery events most likely to be replayed. Either the two numbers are agreed together, or read-time dedupe makes the question moot.

**Ingest QPS — closed, not open.** From the [raw events model](./telemetry-storage-sizing.md#5-volume) (5 auctions/session, `K` = 1): ~0.5 rps at 10k DAU, ~5 rps at 100k, ~50 rps mean at 1M. Size the handler like any other SDK route; no special provision.

### 4.4 Schema validation and rejection

| Condition | HTTP | Event fate |
| --- | --- | --- |
| Invalid JSON / oversize | 413 / 400 | None stored |
| Unknown `schema_version` major | 200 | **Store anyway**, tagged with the version — see below |
| Missing `event_name` / `event_id` | 200, that event rejected | Drop |
| Fields outside the allow-list | 200 | **Drop the fields**, store a count, increment `telemetry_unknown_fields_total` |
| Error events | Accept | Never sampled, never dropped for sampling |

Do not return 4xx for individual bad events inside a valid batch (poison-pill protection for the SDK queue).

**Never hard-reject on `schema_version`.** A version window ("current and previous major") is the conventional choice and is wrong here: PRD rationale 6 states SDK propagation is measured in quarters and never reaches 100%, so rejecting old majors would drop telemetry from a long tail of publishers and bias precisely the fill and reconciliation numbers J3/J4 exist to produce. The lake is schema-on-read — land everything, tag the version, filter in queries.

**Unknown fields are dropped on the client path, not stored.** Keeping them in a bounded `unknown_fields` column is the tempting debug-friendly option, and it contradicts G3: unknown fields on a client batch are exactly where an advertising identifier arrives under a name the denylist does not know. On a public repo, where PRD rationale 5 makes over-collection a liability that cannot be quietly fixed, this is allow-list only. (`adm_parse_result.unknown_fields[]` in the PRD is a different thing — DSP creative-envelope keys observed server-side — and is unaffected.)

### 4.5 Sampling

**Coherent by auction, not independent per event** (requirements G11). Hash `(app_id, auction_id)`; keep or drop that auction's whole diagnostic set.

- **Never sampled, ever:** all errors; every PRD funnel stage (`ad_request_started`, `token_collection_completed`, `auction_requested`, `auction_response_received`, `ad_filled`, `ad_show_requested`, `ad_impression`, `billing_notice_sent`); all notices; `auction_completed`; `dsp_response_rejected`.
- **Sampled as a set** at rate `K` (default 0.1): per-adapter token detail, per-DSP request/response pairs, per-attempt load detail, `renderer_selected` / `ad_loaded`, viewability, VAST quartiles, `adapter_init_result` successes.
- Record `sampling_rate` on every event so `sum(1 / sampling_rate)` corrects counts at query time.
- Config delivers `K`; server-native emits apply the same auction hash, so client and server halves of one auction are kept or dropped together.

The obvious alternative — a sampling rate per `event_name`, applied independently — makes PRD success metric 1 (funnel completeness >98%) unmeasurable by construction, leaves every J1 path with holes, and produces ragged joins where a sampled numerator is divided by an unsampled denominator. Coherence by auction fixes all three at once, and costs nothing beyond hashing an id that is already on the event.

---

## 5. Remote config

Extend `POST /v2/config` response:

```json
{
  "telemetry": {
    "enabled": true,
    "ingest_path": "/v2/telemetry",
    "auction_sample_rate": 0.1,
    "schema_version": 1
  }
}
```

One rate, not a per-`event_name` map: sampling is coherent per auction (§4.5), so a map of independent rates would re-introduce the defect it replaces. Never-sampled classes are a property of the schema, not of config, and cannot be turned off by a mis-set value.

- `enabled: false` is the publisher kill switch.
- Per-auction kill switch (BID-47) can be added later on the auction response; **not** required for M0.
- Storage: start with environment / static defaults, then app-level overrides in DB if publishers need different rates. Exact table is an implementation detail of BE-M0-4.

At the first-iteration design point (~10k DAU) the honest default is **`auction_sample_rate: 1.0`**. This TRD’s lake budget is **unsampled Parquet** (~1.1 GiB/day at 10k). `K` = 0.1 is a later reduction. Ship the knob in M0; turn it down when volume justifies it.

---

## 6. Storage — events (Parquet on Spaces)

This TRD implements **grain A only**.

```
bidon-sdkapi  →  JSON  →  Redpanda telemetry-events
                              →  Parquet sink (flush every 10 min)
                              →  DigitalOcean Spaces
                              →  DuckDB when queried
```

JSON on the topic is the **bus**. **Parquet on Spaces is the store.** Do not land JSON-on-S3. Traces and metrics are out of this implementation. `/v2/show` produces events; it does not query the lake. If the sink or Spaces is down, ads still serve (G1/G2).

Detail: [telemetry-events-store.md](./telemetry-events-store.md). Numbers: [telemetry-storage-sizing.md](./telemetry-storage-sizing.md).

### 6.1 Spaces + sink

| Item | Requirement |
| --- | --- |
| Bucket | DigitalOcean **Spaces** (S3 API), prefix `events/` |
| Format | **Parquet** (snappy or zstd). Envelope columns + `payload` JSON column |
| Layout | `s3://…/events/dt=YYYY-MM-DD/event_name=…/*.parquet` (hive; Iceberg later) |
| **Flush** | **Every 10 minutes**, or **64 MiB** uncompressed batch, whichever first. Do not flush empty. |
| Writer | OSS consumer (Connect S3 parquet or a small writer). One writer. |
| Query | DuckDB `read_parquet` on demand. Not a BI farm in M0. |
| Freshness | Alert if latest object lags the topic by **> 20 min** (two flush intervals) |

10 min lag is acceptable for funnel SQL. Do not page off this lake.

**Small files.** At 10k DAU a flush is ~**41k events / 8 MiB** if one file. Partitioning by `event_name` (~40 names) yields many sub-MiB objects — accepted at 10k; compact or drop `event_name` partitioning if LIST hurts. At 1M a flush is ~780 MiB.

### 6.2 Inputs (assumed until measured)

| Input | Value |
| --- | --- |
| Sessions per DAU | **2** |
| **Auctions per session** | **5** → **10 auctions / DAU** |
| Fill rate | **0.4** |
| Catalog | **23** events/session; **~55** filled / **~54** unfilled auction |
| Parquet | **200 B / row** (unmeasured — ±2×) |
| Sample rate in this table | **`K` = 1.0** |

Campaigns are not a term. Ads served = impressions. Unfilled auctions cost about as much as filled.

```
raw_day     = impressions × 55 + unfilled × 54 + sessions × 23
parquet_day = raw_day × 200 B
```

**590 events / DAU / day.**

### 6.3 Volume (Parquet on Spaces)

| Class | Window | What |
| --- | --- | --- |
| **Aggregates** | **7–10 y** | Daily rollups (date × app × DSP × format × country). Statutory **ledger** — totals, not auction replay. ~25 GiB over seven years. Ship with the sink; cannot backfill. |
| Reconciliation | **400 d** | `ad_impression`, `billing_notice_sent`, `win_notice_sent`, `auction_completed`, `ad_filled` |
| Diagnostics | **90 d** | Everything else, including loss notices |

Tax/accounting does **not** require individual auction runs. 400 d is a 13-month analytics/audit buffer — confirm AppLovin / DSP dispute windows.

| | Auctions / day | Events / day | **Parquet / day** | **On Spaces** (90/400 d) | Per 10 min flush | Spaces $/mo |
| --- | --- | --- | --- | --- | --- | --- |
| **10k DAU** | 100k | **5.9M** | **1.1 GiB** | **114 GiB** | ~41k events, ~8 MiB | **~$5** |
| **100k** | 1M | **59M** | **11 GiB** | **1.1 TiB** | ~410k, ~78 MiB | **~$23** |
| **1M** | 10M | **590M** | **110 GiB** | **11 TiB** | ~4.1M, ~780 MiB | **~$230** |

`1.1 GiB/day` is new Parquet. `114 GiB` is retained. JSON on Redpanda is ~6× larger (~6.8 GiB/day at 10k); keep topic retention in **days**, not 400 d.

Redis 72 h dedupe (client ingest only): **~2.5 / ~25 / ~250 GiB**. Tripwire: SQL `ROW_NUMBER()` at query time before 1M.

### 6.4 Bus

| Topic | Schema | Lifetime |
| --- | --- | --- |
| `ad-events` | current `AdEvent` | Dual-emit until cutover |
| `notification-events` | current `NotificationEvent` | Dual-emit until cutover |
| `telemetry-events` | PRD envelope + body | New — the Parquet sink reads this |

Env: `KAFKA_TELEMETRY_EVENTS_TOPIC`. Create **explicitly** (auto-create is one partition). Set producer **zstd**. Dual-emission is the peak bus window; measure `ad-events` bytes/day before enabling.

### 6.5 Dual-emission and cutover

1. New producers write both buses.
2. Lake queries use Spaces Parquet.
3. Old topics stop after the current consumer owner signs off.
4. `TELEMETRY_EVENTS_ENABLED` gates the new topic only.

---

## 7. Event-pipeline observability

This TRD does **not** implement VictoriaTraces or VictoriaMetrics. Existing `GET /metrics` stays. Auction traces may remain on Sentry until a traces TRD; **do not write spans onto `telemetry-events`**.

Minimum counters on sdkapi:

- `telemetry_ingest_accepted_total{event_name}`
- `telemetry_ingest_duplicate_total`
- `telemetry_ingest_rejected_total{reason}`
- `telemetry_unknown_fields_total`
- `telemetry_produce_failed_total{topic}`
- `telemetry_dropped_total{reason}`
- Sink freshness (latest Spaces object vs topic lag); alert **> 20 min**

Join client and server on **`(app_id, auction_id)`** (G10). `sampling_rate` on every event. `trace_id` on the envelope is reserved for a later traces join; unused by this lake.

Funnel / fill / reconciliation: **DuckDB on Spaces Parquet**.

**PRD client-side loss metric** (`telemetry_dropped` ÷ emitted) has no transport yet — SDK queue, not this sink.

---

## 8. Schema governance

- **Source of truth:** a versioned schema (JSON Schema or proto) that Kotlin SDK and Go server generate from. Until BID-46 lands, Go types in `internal/telemetry` are the backend draft and must not diverge silently.
- `schema_version` integer on every event. M0 starts at `1`.
- Error-code table (PRD bands 1000–11999) lives next to the schema, not hardcoded per handler. Server uses 3000–4999 (request handling, DSP communication) first.
- **Rollout: accept every version, forever.** Tag the row and filter in queries. Hard-rejecting old majors would drop data from publishers on old SDKs — the exact cohort whose fill and reconciliation numbers J3/J4 need — and PRD rationale 6 says that cohort persists for quarters and never empties. The lake is schema-on-read; this costs nothing.
- `dsp_id` vs today's `demand_id`: map 1:1 for adapters; keep `demand_source` as the PRD field; do not invent a second identifier.
- **`sampling_rate` is an envelope field** (PRD). `trace_id` may be present for a later traces join; this TRD does not store or query traces.

---

## 9. Privacy enforcement points

Apply on the **new stream only**. Do not quietly rewrite `ad-events`.

| Point | Action |
| --- | --- |
| Ingest (client batches) | **Allow-list.** Fields not in the schema are dropped, counted, and never stored. A denylist cannot catch an identifier arriving under an unexpected name |
| Ingest | Drop `idfa` / `idfv` / `idg` / `ip` / `city` if present; fill `country` from MaxMind only |
| Server emit | Never copy `raw_request` / `raw_response` onto telemetry events |
| `error_message` | Truncate (256 chars), strip emails/IPs/query strings. Regex scrubbing of free text is best-effort — prefer structured `error_code` and treat the message as a diagnostic bonus, not a field to rely on |
| COPPA (`coppa=true`) | Reduced field set: no advertising ids, no device model if policy requires, keep coarse `os` + `country` |
| LMT / consent | GAID not on this stream at all in M0 (PRD: coarse or null; backend chooses **null**) |
| Notices | Store `notice_type` + `http_status`, not the expanded URL with macros |

Legal review of retention and COPPA field set is required before launch, not after.

**Blocked on a missing PRD section.** The PRD envelope routes all five regulation fields (`gdpr_applies`, `has_tcf_string`, `us_privacy`, `coppa`, `lmt`) to "Note 2", and PRD v1 ends mid-sentence at `Note 2(` with no content. G3 and the COPPA reduced field set are specified against a note that does not exist. This is the highest-priority PRD gap for BE-M0-5.

**Ask legal two questions, not one.** Beyond retention: is `session_id` + `device_model` + `app_bundle` + `country` pseudonymous personal data? If yes, and DSRs must be honoured, row-level delete becomes an M0 requirement — hive-partitioned Parquet cannot do it and Iceberg stops being a Phase 2 nicety. That answer changes the events-store sequencing, so it is worth asking before Phase 1's layout is committed.

Consent provenance (`consent_source`) is an envelope field supplied by the SDK. Server does not infer it, and cannot verify it.

---

## 10. PRD requirement traceability (infrastructure)

| PRD requirement | TRD section | Status |
| --- | --- | --- |
| Common envelope | Schema governance; BE-M0-1 | Draft; blocked on BID-46 |
| Country from IP | Ingest 4.2; privacy | MaxMind already exists |
| Fire-and-forget server emit | Design constraints; 4.1 | Same as current logger |
| `ad_impression` delivery guarantee | §3 | `/v2/show` |
| `event_id` dedupe, burst ingest | §4.3 | Redis 72h, client path only; **QPS closed** (~0.5 rps at 10k raw) with a tripwire to read-time dedupe |
| Sampling + rate on event | §4.5, §5 | Config block new; **auction-coherent** (G11) |
| Kill switch | §5 | Publisher-level in M0 |
| OTel one trace / DSP span | — | **Out of this TRD** (grain B) |
| Metrics vs warehouse | §7 | Funnel SQL on Spaces Parquet. `/metrics` client stays; no VM/VT in this implementation |
| Warehouse + monthly cost | §6 | **Parquet on Spaces, 10 min flush.** 10k: 1.1 GiB/day, 114 GiB retained, ~$5/mo. **Sink/bucket owner still open** |
| No PII, COPPA reduction | §9 | New stream only |
| Schema version, additive changes | §8 | |
| Notice reliability | §3, BE-M1-3 | Sender must record HTTP status |
| Client disk queue / crash path | SDK BID-47 | Out of this TRD’s implementation |
| Reconciliation of old vs new stream | §6.5 | Dual-emission |
| Funnel completeness > 98% | §4.5 | **Requires** auction-coherent sampling. Unmeasurable under independent per-event sampling |
| Telemetry loss rate < 1% | §7 | **No transport defined.** Client-queue counter; needs a PRD/SDK decision |
| Notice delivery rate | §3 | **Not measurable today** — `EventSender` swallows HTTP status. BE-M1-3 |
| COPPA reduced field set | §9 | **Blocked**: PRD "Note 2" is missing |
| Ad-path latency gate | Open item 8 | **Blocked**: PRD target is "N/A" and the risk row's mitigation cell is corrupted |

Any PRD infrastructure dependency not listed here is an open item, not an implicit decision.

---

## 11. Open items

Nothing below is architecture-blocked. Every item needs a **name**.

| # | Item | Blocks | Owner |
| --- | --- | --- | --- |
| 1 | DigitalOcean Spaces bucket + Parquet sink (**10 min flush**) | Grain A lake | **unnamed** |
| 2 | Traces / metrics (VT, VM, otelcol) | Not this TRD | — |
| 3 | Current `ad-events` / `notification-events` consumer | Cutover | **unnamed** |
| 4 | BID-46 envelope sign-off | All `internal/telemetry` types | Tiziano, Jonathan |
| 5 | Legal: retention window **and** whether `session_id` is pseudonymous PII | Launch; possibly Iceberg-in-M0 | **unnamed** |
| 6 | PRD: missing Note 2, corrupted latency risk row, "N/A" latency target, loss-metric transport | BE-M0-5; G1 acceptance | PRD owner |
| 7 | SDK: is `/v2/stats` `ad_units[]` attempt-ordered? | Fill rate, time to fill (BE-M1-6) | SDK |
| 8 | Server-minted `auction_id` (PRD Note 1 vs required request field) | Correlation contract; deferrable | SDK + backend |
| 9 | Impression definition vs OM viewable | Meaning of `ad_impression`, not the channel | OM spike |
| 10 | Measured events per auction and bytes/row | Turns the model into a bill | Backend, post-M0 |

Not this TRD: VictoriaTraces, VictoriaMetrics, Grafana. ClickHouse is events Phase 4 only.

Ticket list and sequencing: [telemetry-m0-m1-backend-spike.md](./telemetry-m0-m1-backend-spike.md#7-m0--m1-backend-tickets).
