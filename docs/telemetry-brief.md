# BidOn Telemetry — backend brief

**Status:** Draft for senior tech / leadership review · **Date:** 2026-08-26 · **Scope:** M0/M1, backend only

Start here. The supporting documents are linked at each decision.

| | |
| --- | --- |
| Why | [PRD v1](./PRD_BidOn_Telemetry%20-%20v1.pdf) (Jonathan Kuperberg) |
| What must be true | [telemetry-requirements.md](./telemetry-requirements.md) |
| Events warehouse (Parquet on Spaces) | [TRD_BidOn_Telemetry.md](./TRD_BidOn_Telemetry.md) |
| What the code does today | [telemetry-m0-m1-backend-spike.md](./telemetry-m0-m1-backend-spike.md) |
| First implementable slice | [telemetry-poc/](./telemetry-poc/README.md) (6 Linear issues, 5 MRs) |
| Stores | [events](./telemetry-events-store.md) · [traces](./telemetry-traces-store.md) · [metrics](./telemetry-metrics-store.md) |
| Numbers | [telemetry-storage-sizing.md](./telemetry-storage-sizing.md) |
| Option history | [telemetry-storage-recommendation.md](./telemetry-storage-recommendation.md) |

---

## 1. The goal

Approve a **server-first first iteration**, and put names against eight owner gaps.

This is a first iteration sized for where we are, with written tripwires for where we're headed.

---

## 2. Where we are

Verified against the code, not asserted:

- **We already emit a production analytics stream** to Redpanda (`ad-events`, `notification-events`) from `bidon-sdkapi`. **This is a migration, not a greenfield.** Any plan that assumes we're starting from zero has the wrong timeline.
- **That stream cannot become the telemetry product.** It carries `idfa`, `idg`, `idfv`, `ip`, `city`, `user_agent`, and full DSP `raw_request` / `raw_response` payloads. It has no `event_id`, no `schema_version`, and no banded error model. Retrofitting the PRD envelope onto it would fuse two incompatible privacy policies into one topic.
- **Traces go to Sentry only** (`config/otel.go`). Sentry is not a queryable trace store.
- **`/metrics` is exposed and stored nowhere.** No Prometheus server, no Grafana, no TSDB in any compose file. A scrape endpoint is not a requirement satisfied.
- **We cannot currently tell whether a billing notice was delivered.** `internal/notification/event_sender.go:52-60` returns `nil` for *any* HTTP response — only transport errors are caught. A 500 from a DSP's BURL endpoint is recorded as success.

That last point is the one to sit with. **PRD Goal 3 — reconcile impressions with mediator reporting within an agreed tolerance — is not achievable today, and the fix is small.**

Adjacent, on the same code path: `internal/notification/handler.go:234` picks the bid to bill using float price equality (`bid.Price == impression.GetPrice()`), first match wins if two DSPs bid identically. Instrumenting reconciliation will surface this. Better fixed deliberately than found during certification.

---

## 3. What we're building

Three kinds of data. They can share infrastructure; they must not share grain.

**A — Product events.** One row per thing that happened. Protobuf → Redpanda `telemetry-events` → Kafka Connect Parquet on Spaces → DuckDB when someone has a question. Iceberg when a catalog exists; ClickHouse later as an accelerator on the *same files*. This is the warehouse: fill rate, render rate, per-DSP loss, reconciliation.

**B — Traces.** One tree per auction, child span per DSP. OTLP → otelcol-contrib → VictoriaTraces. Answers "show me auction X's legs." Sentry stays exceptions-only.

**C — Operational metrics.** Scrape `/metrics` + collector spanmetrics → VictoriaMetrics → vmalert → Alertmanager. Answers "is it bad right now." Never carries `auction_id` on a series.

**Why three and not one.** A self-hosted single-node ClickHouse would pass our own funnel and point-get tests — [the history doc](./telemetry-storage-recommendation.md) makes that case honestly. We preferred the split for grain fit, PromQL-native paging, and blast-radius isolation: a stalled sink cannot stop paging, and a heavy funnel query cannot starve alerts. **This is a judgement call, not something the requirements forced**, and the reversal path (ClickHouse on the same Parquet files) is written down and cheap.

---

## 4. What it costs

At the design point (~10k DAU, **2 sessions × 5 auctions**, fill 0.4, **Parquet on Spaces**, 10 min flush):

| | Per day | Retained (90/400 d) |
| --- | --- | --- |
| **Events (Parquet)** | **1.1 GiB** (5.9M events) | **114 GiB** (~$5/mo Spaces) |

JSON/protobuf on Redpanda is the bus; topic retention is days, not 400 d. Traces and metrics are not in the events TRD. [TRD §5](./TRD_BidOn_Telemetry.md#5-volume-and-cost).

**At 10k, Parquet on Spaces is ~$5/mo** (114 GiB). At 1M: **~11 TiB retained (~$230/mo)**. Redis 72 h dedupe is ~2.5 GiB now and ~250 GiB RAM at 1M unsampled — tripwire to SQL dedupe.

1. **Retention policy.** Statutory 6–10 year accounting applies to the *ledger* (aggregates, ~25 GiB over seven years), not individual auction runs. 400 d on raw reconciliation rows is analytics/audit, not tax. Build the aggregate in M0; it cannot be backfilled.
2. **Redis dedupe at scale.** Fine now (~2.5 GiB). Fallback is read-time dedupe in SQL.
3. **People-time** to run the sink. Owner still unnamed.

---

## 5. Where we're headed

Every component is adequate now and has a written condition for when it isn't. Nothing needs pre-building.

| Component | Good through | Tripwire | Next move |
| --- | --- | --- | --- |
| Parquet + DuckDB | ~100k DAU | Daily/hourly funnel queries, or >2 concurrent SQL users | ClickHouse on the same files |
| Hive Parquet | Until legal rules on DSRs | Row-level delete required | Iceberg + REST catalog |
| Redis dedupe | ~100k DAU | RAM cost exceeds the lake's | Read-time dedupe in SQL |
| Trace sampling | 100% keep to ~100k DAU | Local disk pressure | Step to 10%, errors always kept |
| Single-node VM / VT | Past 1M DAU / ~1M DAU | Scrape fan-out; lookup latency | vmagent, then cluster |
| Engineer-mediated SQL | M0/M1 | BD/support asking weekly | A query surface — M2 conversation |

---

## 6. Sequencing — the main proposal

The PRD's strategic drivers are certification reconciliation and per-DSP delivery evidence. **Both are server-owned with zero SDK dependency.**

The intuitive order — build the ingest endpoint and sampling config first, since they are numbered M0 — puts the SDK-coupled work at the front. That work is blocked on BID-46 and produces nothing until an SDK ships and publishers upgrade, which the PRD itself says takes *quarters*.

**Phase A — foundation.** Envelope types, `telemetry-events` topic, privacy enforcement, dual-write flag.

**Phase B — server events. This is the demo.** Notice outcomes first (unblocks Goal 3), then DSP fan-out outcomes, `auction_completed`, `/v2/show` impression, OTel spans.

> At the end of Phase B, with **Go changes only** — no client transport, no ingest endpoint, no publisher upgrade cycle — Bidon can answer: per-DSP bid / nobid / timeout rate, notice delivery rate, clearing price vs floor, and where the auction latency budget goes.

**Phase C — client ingest.** `POST /v2/telemetry`, `/v2/config` telemetry block, waterfall/fill. Gated on BID-46 / BID-47 and the SDK's confirmation that `/v2/stats` `ad_units[]` preserves attempt order.

Phases B and C are independent once A lands, and can run in parallel given capacity.

---

## 7. What needs to be owned


| # | Owner gap | Blocks |
| --- | --- | --- |
| 1 | Object-storage bucket + Parquet sink | Grain A entirely |
| 2 | VictoriaMetrics / VictoriaTraces / otelcol in Compose + Coolify | Grains B and C |
| 3 | Who consumes `ad-events` / `notification-events` today | Cutover, and the dual-emission cost window |
| 4 | BID-46 envelope sign-off | All backend types |
| 5 | Legal + commercial: statutory window (ledger), **contractual** dispute/audit window (events), and whether `session_id` is pseudonymous PII | Launch; possibly Iceberg-in-M0 |
| 6 | PRD owner: four gaps in §8 below | Privacy spec; G1 acceptance |
| 7 | SDK: is `/v2/stats` `ad_units[]` attempt-ordered? | Fill rate, time to fill |
| 8 | Backend: measure events/auction and bytes/row post-M0 | Turns the model into a bill |

---

## 8. What we're assuming, and what's still open

**Load-bearing assumptions.** Two numbers scale the entire lake linearly and neither is measured: **1.2 KiB per event on the bus** and **200 B per row in Parquet**. Treat every storage figure as ±2× until Phase 1 produces real bytes. The 200 B is the more optimistic of the two — envelope columns dictionary-encode well, but the `payload` JSON blob does not.

The event model itself (~55 raw / ~21.6 stored per auction) is counted from the PRD catalog, not measured from traffic. It sits above the PRD's own "20–40× per impression" estimate, which is why sampling is load-bearing rather than optional. Measuring `event_name` counts per auction is the first thing to do after M0.

**Non-obvious design choices worth surfacing.** Sampling is coherent per auction rather than per event — the conventional per-`event_name` approach would make PRD success metric 1 unmeasurable by construction. All joins are scoped `(app_id, auction_id)` because `auction_id` is client-supplied on a public open-source SDK and is not globally unique. Unknown fields on client batches are dropped rather than kept for debugging, since that is exactly where an identifier arrives under an unexpected name. Schema version is recorded but never a rejection reason.

**The October 14 PRD is the product spec.** `Envelope` carries its common header. Fields the request has not produced stay at the proto3 default. `advertising_id` is always present: the real id only when the PRD table says so, otherwise the all-zero UUID, with `advertising_id_state` recording which rule matched (COPPA, no consent, limit-ad-tracking, or real).

`schema_version` is the PRD int. The current major is `1`. Additive changes stay on that number.

Two limits are technical, and the target remains the PRD:

- `auction_id` is the id the API requires the client to send. The server echoes it. Minting a replacement is a breaking SDK change, and PRD rationale 6 says those upgrades take quarters. Until the request stops requiring the client id, joins stay `(app_id, auction_id)`. `app_id` stays because that client id is not globally unique. `trace_id` stays so the event joins the PRD's per-auction trace. `bundle` is the PRD `app_bundle`.
- Sampling is one rate per auction, keyed on `(app_id, auction_id)`, so PRD success metric 1 (funnel completeness) can be computed. Errors, wins, billing notices, and `ad_impression` are never sampled. Funnel stages stay in that set too: dropping them on their own would leave holes in the same metric. The POC emits `sampling_rate = 1.0`.

`sdk_uptime_ms`, GPP, `mspa_enforced`, `dnt`, `has_tcf_string`, `consent_source`, `os_api_level`, `publisher_id`, and `ad_unit_id` are on the message and left unset. The auction request does not carry them. The session clock is a monotonic tick, so it is not written into `sdk_uptime_ms`.

**Still open with the PRD owner:**

- COPPA reduced field set. Note 2 now exists (consent fields are passthrough, `consent_source` records who supplied them) and it does not list which fields drop. The only concrete rule is that `advertising_id` becomes an all-zero UUID.
- `sampling_rate` is on the envelope. The type cell says `init`. Treat it as `double`.
- `telemetry_dropped` is a client queue counter (disk-backed queue, oldest-first eviction). It is not a catalog event. The server counter `telemetry_dropped_total` is a different quantity.
- Define `demand_load_attempt` and `waterfall_exhausted`, or drop them. Fill rate is defined on `ad_filled`, which the proto has and the M1 tables never specify. Token events are in the proto and the M3 migration table, and missing from the M1 tables.

**Unresolved by choice.** Monthly cost stays a range until volume is measured. The three-store split is a preference we can defend but did not derive from the requirements — [the option history](./telemetry-storage-recommendation.md) holds the counter-case. And J3/J4/J5 name non-engineers as users while M0/M1 delivers engineer-mediated SQL; that expectation needs setting with BD and support directly.
