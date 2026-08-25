# Telemetry storage sizing

Planning model for [events](./telemetry-events-store.md), [traces](./telemetry-traces-store.md), and [metrics](./telemetry-metrics-store.md).

**Design point is the first iteration, not the ceiling.** The tables below lead with ~10k DAU (where we are), then show 100k and 1M as headroom checks. Nothing here is sized for 10M DAU on day one, and nothing here has to be — the tripwires in §7 say when each component stops being adequate.

Replace inputs with measured direct-mode traffic before using any figure as a bill. §8 lists what to measure.

---

## 1. Inputs

| Input | Symbol | Default | Confidence |
| --- | --- | --- | --- |
| Sessions per DAU | `S` | 2 | Assumed |
| Auctions per DAU | — | 4 (8 at the 1M tier) | Assumed |
| Registered network adapters | `A` | 20 | PRD ("20+ network adapters") |
| Bidding DSPs per auction | `D` | 6 | Assumed |
| DSPs that return a bid | `B` | 4 | Assumed |
| Waterfall attempts, filled | `W` | 4 | Assumed |
| Waterfall attempts, unfilled | `Wᵤ` | 6 (exhausted) | Assumed |
| Loss notices | `L` | `B−1` = 3 filled, `B` = 4 unfilled | Derived |
| Video share of impressions | `V` | 0.5 | Assumed |
| Fill rate | — | 0.4 | Assumed |
| **Auction sample rate** | `K` | **1.0** at the design point, **0.1** at scale (see §3) | Policy |
| Event JSON on the bus | — | **1.2 KiB** | **Unmeasured — highest leverage** |
| Event row in Parquet | — | **200 B** | **Unmeasured — highest leverage** |
| Trace span in VictoriaTraces | — | 150 B | Vendor claim |
| Metric sample in VictoriaMetrics | — | 1 B + 20% index + 25% merge | Vendor claim |
| Redis `tel:evt:{app_id}:{event_id}` | — | 200 B, 72 h | Estimated |

`K` = 0.1 is used in §3–§4 to derive per-unit figures. **At the ~10k DAU design point the operational default is `K` = 1.0** — 472 MiB/day is not worth sampling away, and full fidelity while calibrating is worth more than the storage. §5 tiers each state the `K` they use.

The two "highest leverage" rows scale the entire lake linearly. 200 B/row assumes columnar dictionary + zstd on a mostly-repetitive envelope; the `payload` JSON blob column compresses far worse than the envelope columns, so **200 B is optimistic until measured**. Treat lake figures as ±2×.

---

## 2. Event catalog (derived from PRD Part II)

Counted from the PRD's Part II catalog rather than from a representative subset, one auction emits **~55 raw events** — above the PRD's own "20–40× per impression" hypothesis. The gap comes from families that are easy to overlook because each is individually small: `adapter_init_result` across 20+ registered adapters, `config_fetch_result`, the `adapter_token_requested` / `adapter_token_result` pair, `token_collection_completed`, the four client-side auction events, `adm_parse_result` per attempt, `renderer_selected`, `ad_loaded`, the display block, and all seven VAST quartile events.

That number sizes the **bus, the client queue and ingest**. It is not what sizes the lake — see §3.

### Per session (app launch)

| Event | Count |
| --- | --- |
| `sdk_init_started`, `sdk_init_completed` | 2 |
| `adapter_init_result` | `A` = 20 |
| `config_fetch_result` | 1 |
| **Raw per session** | **23** |

### Per auction

| Stage | Filled | Unfilled |
| --- | --- | --- |
| `ad_request_started` | 1 | 1 |
| `adapter_token_requested` + `adapter_token_result` | 2`D` = 12 | 12 |
| `token_collection_completed` | 1 | 1 |
| `auction_requested` + `auction_response_received` | 2 | 2 |
| `auction_request_received` + `auction_completed` (server) | 2 | 2 |
| `dsp_request_sent` + `dsp_response_received` (server) | 2`D` = 12 | 12 |
| `dsp_response_rejected` (server) | 1 | 1 |
| `ad_load_requested` + `adm_parse_result` | 2`W` = 8 | 2`Wᵤ` = 12 |
| `ad_load_failed` | `W−1` = 3 | `Wᵤ` = 6 |
| `renderer_selected` + `ad_loaded` + `ad_filled` | 3 | 0 |
| `auction_no_demand` / `waterfall_exhausted` | 0 | 1 |
| Display (`ad_show_requested`, `ad_impression`, `ad_viewable`, `ad_closed`) | 4 | 0 |
| VAST quartiles (`V` × 6) | 3 | 0 |
| Notices (`win` + `billing` + `L`) | 5 | 4 |
| **Raw per auction** | **~55** | **~54** |

Note that unfilled auctions are **not** cheaper than filled ones. A waterfall that exhausts generates more load-attempt and error events than one that fills on the fourth try — it loses the display and video blocks but gains attempts and failures. Modelling no-fill as the cheap case would understate the total, since it is the majority case at a 0.4 fill rate.

---

## 3. Sampling: coherent by auction, not independent per event

Independent per-event sampling breaks two things the PRD asks for:

- **Funnel completeness** (PRD success metric 1, target >98%) counts requests with an *unbroken* event chain. If any funnel stage is sampled independently, essentially no request has an unbroken chain.
- **Ragged joins.** Sampling `dsp_response_received` at 0.1 while `dsp_request_sent` is unsampled makes bid rate `(numerator × 10) ÷ denominator` — correct in expectation, fragile in practice, and useless for reconstructing one auction (J1).

**Policy: hash `(app_id, auction_id)` and keep or drop the whole auction's diagnostic set.** Two tiers:

**Tier 1 — always kept, every auction.** The PRD funnel stages, every error, every notice, impression and billing. These define fill rate, render rate, and reconciliation, so they are never sampled and never version-rejected.

| Always kept | Filled | Unfilled |
| --- | --- | --- |
| `ad_request_started`, `token_collection_completed` | 2 | 2 |
| `auction_requested`, `auction_response_received` | 2 | 2 |
| `auction_request_received`, `auction_completed` | 2 | 2 |
| `ad_filled` / `auction_no_demand` | 1 | 1 |
| `ad_load_failed` (errors) | 3 | 6 |
| `dsp_response_rejected` (errors) | 1 | 1 |
| `ad_show_requested`, `ad_impression` | 2 | 0 |
| Notices (`win` + `billing` + `L`) | 5 | 4 |
| **Subtotal** | **18** | **18** |

**Tier 2 — kept on sampled auctions only** (rate `K`): per-adapter token detail, per-DSP request/response, per-attempt load detail, `renderer_selected` / `ad_loaded`, viewability, VAST quartiles. ~36 filled / ~30 unfilled.

**Session events:** init *failures* are errors and always kept; `adapter_init_result` successes are Tier 2. Always kept ≈ 4/session; sampled ≈ 19/session.

### Stored per unit, at `K` = 0.1

| Unit | Stored |
| --- | --- |
| Filled auction | 18 + 0.1 × 36 = **21.6** |
| Unfilled auction | 18 + 0.1 × 30 = **21.0** |
| Session | 4 + 0.1 × 19 = **5.9** |

At `K` = 0.1 that is a ~2.6× reduction against the ~55 raw events per auction.

**Sampling is applied at emit, so `K` reduces everything downstream of it** — the client queue, the ingest endpoint, Redpanda and the lake alike. The raw count in §2 is what the instrumentation produces before that gate; it equals transported volume only at `K` = 1. Sizing the bus from the raw count while sizing the lake from the stored count would overstate Redpanda by the sampling factor.

---

## 4. Retention

### Three drivers, not one

Retention questions collapse if you treat "how long must we keep bidding data" as a single number. It is three:

| Driver | Applies to | Typical window | Who owns it |
| --- | --- | --- | --- |
| **Statutory accounting / tax** | The **ledger**: invoices, settlement statements, aggregate revenue per counterparty per period | **6–10 years** by jurisdiction (US commonly 7; UK 6; DE/FR up to 10) | Finance + counsel |
| **Contractual dispute / audit** | **Impression-level** rows a partner could challenge | Dispute windows commonly 60–180 d; audit rights often 12–24 months | Commercial / the MAX certification agreement |
| **Privacy (storage limitation)** | Anything pseudonymous — `session_id`, device, bundle | As short as defensible; GDPR Art. 5(1)(e) makes *longer is safer* false | Counsel + DPO |

**No tax authority requires a per-impression event log.** What substantiates an invoice is the aggregate you billed, not the ability to replay every auction. Keeping raw events for seven years would be ~6× the 400-day figure (~25 TiB at 1M DAU) to hold data that is weaker evidence than a signed aggregate and carries live privacy exposure.

The binding constraint on **event-level** retention is therefore the contractual window, which is unchecked. Confirm it against the AppLovin certification terms and standing DSP agreements before treating 400 d as settled.

### Classes

| Class | Window | Contents |
| --- | --- | --- |
| **Aggregates** | **7–10 y** (match the statutory window) | Daily rollups: date × app × DSP × ad_format × country → impressions, billable amount, notice successes. No ids, no per-user grain |
| Reconciliation set | **400 d** | `ad_impression`, `billing_notice_sent`, `win_notice_sent`, `auction_completed`, `ad_filled` |
| Diagnostics | **90 d** | Everything else, including loss notices |

The 400 d figure is a **13-month analytics rationale** (year-over-year plus an audit buffer), not a legal derivation. Say so when quoting it.

Loss notices sit at 90 d deliberately: only events evidencing a *billing* claim need to survive a reconciliation dispute; loss notices are DSP optimisation signal. Keeping the reconciliation class narrow is worth roughly 20% of the lake.

### Aggregates are effectively free

A daily rollup at ~50k rows/day × 200 B is ~10 MB/day, so **seven years is ~25 GiB** — under 1% of the 1M-DAU lake, and independent of `K` because it is computed from never-sampled events. Even a fat dimension set (adding placement, or full country granularity) lands in the low hundreds of GiB.

This is the cheapest insurance in the whole design: it satisfies the statutory driver, survives any future decision to shorten raw-event retention on privacy grounds, and costs a scheduled DuckDB query writing one small Parquet file per day. Build it in M0 rather than reconstructing history later — the aggregate cannot be backfilled once the raw rows expire.

**Sensitivity on the raw classes.** At the 1M tier: 400 d → 180 d cuts retained bytes ~30%; 90 d → 30 d cuts ~45%. Both are live options once the aggregate tier exists.

Retained bytes per unit at steady state (200 B/row):

At `K` = 0.1:

```
filled auction   = 5 × 200 B × 400 d + 16.6 × 200 B × 90 d = 698,800 B = 0.67 MiB
unfilled auction = 1 × 200 B × 400 d + 20.0 × 200 B × 90 d = 440,000 B = 0.42 MiB
session          =                     5.9 × 200 B × 90 d  = 106,200 B = 0.10 MiB
```

At `K` = 1.0 (the design-point setting). The 400-day class is never sampled, so only the 90-day term grows:

```
filled auction   = 5 × 200 B × 400 d + 49 × 200 B × 90 d = 1,282,000 B = 1.22 MiB
unfilled auction = 1 × 200 B × 400 d + 47 × 200 B × 90 d =   926,000 B = 0.88 MiB
session          =                     23 × 200 B × 90 d =   414,000 B = 0.39 MiB
```

Binary units throughout (KiB = 1024 B, MiB = 1024 KiB), so these do not match a decimal reading of the same byte counts.

---

## 5. Volume

The unit of account is the **auction**, not the ad. At fill 0.4 most auctions never produce an ad, and an unfilled auction costs about as much as a filled one. Sessions are a **separate term** because init events scale with app launches, not auctions. And `K` multiplies **only the diagnostic tier** — the ~18 always-kept events per auction are unaffected by it.

```
sessions_day    = DAU × S
auctions_day    = DAU × auctions_per_DAU
impressions_day = auctions_day × fill
unfilled_day    = auctions_day − impressions_day

stored_day  = impressions_day × (18 + K × 36)
            + unfilled_day    × (18 + K × 30)
            + sessions_day    × ( 4 + K × 19)

parquet_day = stored_day × 200 B
bus_day     = stored_day × 1.2 KiB       ← sampling is applied at emit
s3_retained = impressions_day × r_filled + unfilled_day × r_unfilled + sessions_day × r_session
traces_day  = auctions_day × trace_keep × 1.5 KiB
redis       = client_events_day × 200 B × 3 d
```

**Sampling happens before the bus, not after.** Both producers apply `K` at emit (TRD §4.5), so Redpanda, the client queue and the ingest endpoint all carry *stored* volume, not raw. The ~55 raw events per auction from §2 is the instrumentation's full output — it equals the transported volume only at `K` = 1.

`r_*` are the retained-bytes-per-unit figures from §4, which depend on `K`. They are quoted below per tier rather than as constants.

### Tier 1 — first iteration (10k DAU, 4 auctions, `S`=2, fill 0.4, **`K` = 1.0**)

Full fidelity, because at this volume sampling buys nothing worth the loss of detail.

```
auctions 40k · impressions 16k · unfilled 24k · sessions 20k
stored_day   = 16k×54 + 24k×48 + 20k×23   = 2.48M events/day
parquet_day  = 2.48M × 200 B              = 472 MiB / day
bus_day      = 2.48M × 1.2 KiB            = 2.8 GiB / day (≈ 0.8 GiB zstd)
s3_retained  = 16k×1.22 + 24k×0.88 + 20k×0.39 MiB  ≈ 47 GiB
traces_day   @ 100% keep                  = 59 MiB / day → 820 MiB at 14 d
redis (72 h) = 1.7M client events × 200 B × 3       ≈ 1.0 GiB
ingest       = 1.7M / 100 per batch                 ≈ 0.2 rps mean
```

### Tier 2 — growth (100k DAU, 4 auctions, `S`=2, fill 0.4, `K` = 0.1)

```
auctions 400k · impressions 160k · unfilled 240k · sessions 200k
stored_day   = 160k×21.6 + 240k×21.0 + 200k×5.9  = 9.68M events/day
parquet_day  = 1.8 GiB / day
bus_day      = 11.1 GiB / day (≈ 3 GiB zstd)
s3_retained  = 160k×0.67 + 240k×0.42 + 200k×0.10 MiB  ≈ 223 GiB
traces_day   @ 100% keep = 572 MiB / day → 7.8 GiB at 14 d
redis (72 h) = 5.8M client events × 200 B × 3    ≈ 3.2 GiB
ingest       ≈ 0.7 rps mean
```

### Tier 3 — headroom check (1M DAU, 8 auctions, `S`=3, fill 0.4, `K` = 0.1)

```
auctions 8M · impressions 3.2M · unfilled 4.8M · sessions 3M
stored_day   = 3.2M×21.6 + 4.8M×21.0 + 3M×5.9    = 187.6M events/day
parquet_day  = 35 GiB / day
bus_day      = 215 GiB / day (≈ 57 GiB zstd)
s3_retained  = 3.2M×0.67 + 4.8M×0.42 + 3M×0.10 MiB    ≈ 4.25 TiB
traces_day   @ 10% keep = 1.1 GiB / day → 16 GiB at 14 d
redis (72 h) = 110M client events × 200 B × 3    ≈ 62 GiB   ← see §7, tripwire
ingest       ≈ 13 rps mean, ~65 rps peak
```

Tier 1 runs at `K` = 1.0 and Tiers 2–3 at `K` = 0.1, which is the realistic operating plan rather than a like-for-like comparison. Holding Tier 1 at `K` = 0.1 would give 185 MiB/day and ~22 GiB retained — the floor, not the plan.

### Summary

| | `K` | Stored / day | Parquet / day | Retained on S3 | Bus / day (zstd) | Traces on VT (14 d) | Redis 72 h | Ingest |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| **10k DAU** | **1.0** | 2.5M | **472 MiB** | **47 GiB** | 0.8 GiB | 820 MiB | 1.0 GiB | 0.2 rps |
| 100k DAU | 0.1 | 9.7M | 1.8 GiB | 223 GiB | 3 GiB | 7.8 GiB | 3.2 GiB | 0.7 rps |
| 1M DAU | 0.1 | 188M | 35 GiB | 4.25 TiB | 57 GiB | 16 GiB | **62 GiB** | 13 rps |

Metrics are independent of DAU — see §6.

Every figure here is post-sampling. The ~55 raw events per auction in §2 sizes the instrumentation, not the pipeline; it equals the transported volume only at `K` = 1, which is why Tier 1's bus figure is the largest relative to its DAU.

---

## 6. Metrics, derived

Series count is a function of instances and label cardinality, not ad volume:

| Source | Series / instance |
| --- | --- |
| Go runtime + process | ~120 |
| `echoprometheus` (routes × status class × histogram buckets) | ~300 |
| Telemetry counters + `dsp_request_duration` histogram | ~450 |
| Collector spanmetrics | ~400 |
| **Per instance** | **~1,300** |

× 3 `sdkapi` replicas + 1 `admin` ≈ **~5,000 series**. This multiplies by replica count — a scale-out doubles it.

```
5,000 × (86400 / 15) × 1 B × 90 d × 1.20 × 1.25  ≈  3.6 GiB
```

**~4 GiB on one disk, at any DAU.** Grain C is not a cost conversation.

---

## 7. Tripwires

Each component is adequate now with a written condition for when it stops being.

| Component | Adequate through | Tripwire | Next move |
| --- | --- | --- | --- |
| Parquet + DuckDB | ~100k DAU | Funnel queries daily/hourly, or >2 concurrent SQL users | ClickHouse on the **same** files (events Phase 4) |
| Hive-partitioned Parquet | Until legal rules on DSRs | Row-level delete required, or file counts explode | Iceberg + REST catalog (events Phase 2) |
| **Redis `event_id` dedupe** | ~100k DAU (4 GiB) | RAM cost exceeds the lake's | Read-time dedupe: `ROW_NUMBER() OVER (PARTITION BY event_id)`. Lake is append-only and every query is batch, so G5 holds logically without RAM |
| Single-node VictoriaTraces | ~1M DAU at 10% keep | Ingest or lookup latency | VT cluster |
| Single-node VictoriaMetrics | Well past 1M DAU | Scrape fan-out | vmagent, then VM cluster |
| `telemetry-events` partitions | Set explicitly at creation | Auto-created topics default to 1 partition | Size partitions to peak raw/day before first write |

Dedupe scope note: **~15 of the ~55 raw events per auction are server-minted.** A server-generated UUID cannot arrive twice, so it never needs a Redis entry. Dedupe applies to the client ingest path only — the Redis figures above already assume that.

Trace sampling note: at Tier 1 and Tier 2, **keep 100% of traces.** 820 MiB and 7.8 GiB on local disk are not worth sampling away, and a 1% head sample would mean the auction someone escalates is absent 99% of the time — which defeats the reason grain B exists. Drop to ~10% (errors always kept) around Tier 3.

---

## 8. Measure

In priority order. The first two move the lake estimate by more than everything else combined.

1. **Mean JSON bytes on the bus, and Parquet bytes/row** — replaces 1.2 KiB / 200 B.
2. **`event_name` count per `(app_id, auction_id)`** — replaces the ~55 raw / ~21.6 stored model.
3. Histograms of `D`, `B`, `W`, and video share `V`.
4. Sessions per DAU and `adapter_init_result` count per session.
5. Producer compression ratio (set zstd explicitly; do not rely on the franz-go default).
6. VM series count after instrumentation — confirm no `auction_id` label reached a series.
7. VT disk per retained tree — replaces 150 B/span.
