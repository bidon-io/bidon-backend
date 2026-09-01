# Telemetry M0/M1 backend spike

**Status:** Draft (backend audit + ticket list; no production instrumentation)  
**Date:** 2026-08-26  
**Start here:** [telemetry-brief.md](./telemetry-brief.md)  
**Source PRD:** [PRD_BidOn_Telemetry - v1.pdf](./PRD_BidOn_Telemetry%20-%20v1.pdf)  
**Requirements:** [telemetry-requirements.md](./telemetry-requirements.md)  
**Events warehouse TRD:** [TRD_BidOn_Telemetry.md](./TRD_BidOn_Telemetry.md) (Parquet on Spaces only; not ingest, traces, or metrics)  
**Settled stores:** events (A) [telemetry-events-store.md](./telemetry-events-store.md) · traces (B) [telemetry-traces-store.md](./telemetry-traces-store.md) (VictoriaTraces) · metrics (C) [telemetry-metrics-store.md](./telemetry-metrics-store.md) (VictoriaMetrics)  
**Sizing:** [telemetry-storage-sizing.md](./telemetry-storage-sizing.md)  
**Option we declined:** [telemetry-storage-recommendation.md](./telemetry-storage-recommendation.md) (unified ClickHouse)  
**Linear initiative:** [Telemetry M0/M1 foundations](https://linear.app/bidon/initiative/telemetry-m0m1-foundations-validate-assumptions-and-size-the-build-2d2f5f28ed90) (read-only; this spike does not file or comment there)

This document answers the backend-owned spike questions, inventories the event stream that already exists, maps it onto the PRD catalog, and lists M0/M1 backend tickets. SDK envelope/transport review stays with BID-46 / BID-47 / BID-54.

## Verdict

This is a **migration**, not a greenfield ingest.

`bidon-sdkapi` already emits a production analytics stream to Redpanda (`ad-events`, `notification-events`). That stream is not the PRD M0 envelope: it has no `event_id` / `schema_version` / banded error model, it carries PII, and several PRD server events are missing or collapsed into a single `event_type`.

Hard-cutting the current topics would break consumers that live outside this repo. Dual-emit old and new until those consumers migrate.

Production instrumentation is **out of scope** for this spike. Implement after the mapping and TRD are accepted.

## Scope

**In**

- Android-first, matching the PRD; this repo (`bidon-backend`) only
- M0 envelope/ingest/config/OTel skeleton
- M1 server events (auction, DSP fan-out, outbound notices) and the `/v2/show` impression/billing path

**Out**

- Any production event emission
- M2 / M3 event implementation
- Dashboards, alerting product, OM SDK
- Creating or editing Linear issues, comments, or documents

---

## 1. Current pipeline

```
SDK  --POST /v2/{config,auction,stats,show,click,win,loss,reward}-->  bidon-sdkapi
                                                                         |
                                                                         | JSON AdEvent / NotificationEvent
                                                                         v
                                                              Redpanda: ad-events
                                                                        notification-events
                                                                         |
                                                                         v
                                                              consumers outside this repo

bidon-sdkapi  --HTTP GET (3 retries)-->  DSP nurl / burl / lurl
```

Key packages:

| Piece | Location |
| --- | --- |
| Event types + marshal | `internal/sdkapi/event/event.go` |
| Kafka producer | `internal/sdkapi/event/engine/kafka.go`, `config/kafka.go` |
| Auction + DSP events | `internal/auction/service.go` (`logEvents`, `prepareAuctionRequestEvent`, `prepareBiddingEvents`) |
| Domain handlers | `internal/sdkapi/v2/router.go`, `internal/sdkapi/v2/apihandlers/*` |
| Notices | `internal/notification/handler.go`, `internal/notification/event_sender.go` |
| Country from IP | `internal/sdkapi/geocoder`, used in `BaseHandler.resolveRequest` via `c.RealIP()` |
| OTel | Prometheus metrics + Echo/gRPC/GORM; traces exported to Sentry (`config/otel.go`) |

Auth for SDK calls is **app key + bundle** lookup (`AppFetcher.FetchCached`), plus `X-Bidon-Version`. There is no dedicated telemetry credential.

### 1.1 Endpoints that already produce events

| Endpoint | Handler | `event_type`(s) | Side effects |
| --- | --- | --- | --- |
| `POST /v2/config` | `config_handler.go` | `config` | Adapter init payload |
| `POST /v2/auction/{ad_type}` | `auction.Service.Run` | `auction_request`, per-DSP `bid_request`, successful `bid` | Bidding fan-out; Redis auction result for notices |
| `POST /v2/stats/{ad_type}` | `stats_handler.go` | `stats_request`, `demand_request` (CPM), `client_bid` (RTB) | NURL/LURL if `external_win_notifications` is false |
| `POST /v2/show/{ad_type}` | `show_handler.go` | `show` | BURL (billing) for bidding ads |
| `POST /v2/click/{ad_type}` | `click_handler.go` | `click` | None |
| `POST /v2/reward/{ad_type}` | `reward_handler.go` | `reward` | None |
| `POST /v2/win/{ad_type}` | `win_handler.go` | `win` | NURL/LURL if `external_win_notifications` is true |
| `POST /v2/loss/{ad_type}` | `loss_handler.go` | `loss` | LURL if `external_win_notifications` is true |

There is **no** `/v2/telemetry` (or equivalent) batch ingest.

### 1.2 Event inventory

**Topic `ad-events` (`AdEvent`)**

| `event_type` | When | Notable fields beyond the common envelope |
| --- | --- | --- |
| `config` | Config fetch | Status `SUCCESS` |
| `auction_request` | After auction build (also on error, via `defer`) | `status` SUCCESS/ERROR, `error`, `price_floor` |
| `bid_request` | Per DSP after the HTTP call returns | `demand_id`, HTTP `status` as string, `ecpm`, `raw_request`, `raw_response`, `error`, `timing_map.bid` + `timing_map.token` |
| `bid` | Per DSP when `DemandResponse.IsBid()` | `ecpm` = bid price, `timing_map.bid` |
| `stats_request` | Client reports auction outcome | Winner demand/price, `timing_map.auction` |
| `demand_request` | Per CPM ad unit in stats | Unit `status`, fill timings, `error` |
| `client_bid` | Per RTB ad unit in stats | Fill + token timings |
| `show` | Impression callback | Demand, ad unit, price |
| `click` | Click callback | Same bid identity fields |
| `reward` | Reward callback | Same |
| `win` | External win notification path | Same |
| `loss` | External loss path | Plus `external_winner_demand_id` / `external_winner_ecpm` |

Common `AdEvent` fields (always populated from `BaseRequest` + geo): timestamp, device (manufacturer, model, os, os_version, connection_type, device_type, user_agent), session (id, uptime, RAM, battery, CPU, storage, memory-warning timestamps), app (bundle, versions, framework, plugin), identifiers (`idfa`, `idg`, `idfv`, `app_set_id`), regulations (COPPA, GDPR), geo (`country_code`, `city`, `ip`, `country_id`), segment, `ext`, `mediation_mode`, `mediator`.

Missing versus the PRD envelope: `event_id`, `event_name`, `schema_version`, `host`, `auction_owner`, `integration_mode`, `sampling_rate`, `publisher_id`, banded `error_domain` / `error_code`, `demand_type`, OM fields.

**Topic `notification-events` (`NotificationEvent`)**

| `event_type` | OpenRTB meaning | When |
| --- | --- | --- |
| `NURL` | Win notice | Stats SUCCESS (internal path) or `/win` (external path) |
| `LURL` | Loss notice | Below floor; stats FAIL / AUCTION_CANCELLED; `/loss`; losers on `/win` |
| `BURL` | Billing notice | `/v2/show` for a bidding impression |
| `TimeoutURL` | Timeout | DSP deadline exceeded and adapter supplied a timeout URL (Meta today) |

`NotificationEvent` fields: timestamp, event_type, bundle, ad_type, demand_id, auction_id, imp_id, loss_reason, ecpm, first_price, second_price, url, template_url, error. No device, no geo, no `http_status`, no `retry_count`, no `success`, no `latency_ms`.

Retries: exponential backoff, max 3, in `EventSender.SendEvent`. Failure is a log line plus `Error` on the event; there is no `notice_delivery_failed` event.

**Sharper than "no `http_status` field": the HTTP outcome is never examined.**

```52:60:internal/notification/event_sender.go
	err = backoff.Retry(func() error {
		httpResp, err := es.HttpClient.Get(u.String())
		if err != nil {
			log.Printf("SendNotificationEvent: send failed: %v", err)
			return err
		}
		defer httpResp.Body.Close()

		return nil
	}, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3))
```

The closure returns `nil` for **any** HTTP response. Only transport errors are retried or recorded. A 500 from a DSP's BURL endpoint is indistinguishable from a 200.

So the gap is not "the field list is incomplete" — it is that **Bidon cannot currently tell whether a billing notice was accepted.** J4 and PRD Goal 3 (reconcile with mediator reporting within an agreed tolerance) are unachievable until BE-M1-3 lands. That reprioritises BE-M1-3 to the front of M1; see [Suggested sequence](#suggested-sequence).

**Separately: the bid selected for billing is matched by float price equality.**

```233:236:internal/notification/handler.go
	for _, bid := range auctionResult.Bids {
		if bid.Price == impression.GetPrice() {
			go h.Sender.SendEvent(ctx, Params{
```

Float equality across a client round-trip, first match wins when two DSPs bid identically. The billing notice can be attributed to the wrong demand. This is a pre-existing defect, not telemetry work — but it sits directly on the reconciliation path the PRD is built around, and instrumenting that path will surface it. Better fixed deliberately than discovered during certification.

### 1.3 PII on the current stream

The PRD forbids PII on telemetry. Today's `AdEvent` includes:

| Field | Why it is PII / sensitive |
| --- | --- |
| `idfa`, `idfv`, `idg`, `app_set_id` | Advertising / device identifiers |
| `ip` | Client IP from geocoder |
| `city` | Finer than country |
| `user_agent` | Often treated as personal in combination with IP |
| `raw_request`, `raw_response` on `bid_request` | Full DSP payloads; can include user/device data Bidon forwarded |
| Session RAM / battery / storage | Device fingerprinting surface |

`country_code` / `country_id` are server-derived from MaxMind and are allowed (PRD: country is server-side, never from the client).

`NotificationEvent` is thinner but `url` is the fully expanded notice URL (macros substituted), which can carry auction/user identifiers the DSP embedded.

### 1.4 How `mediation_mode` / `mediator` are set

Not enums. They are optional strings copied from request `ext`:

```90:105:internal/sdkapi/schema/base_request.go
func (r *BaseRequest) GetMediationMode() string {
	ext := r.GetExtData()
	if mode, ok := ext["mediation_mode"].(string); ok {
		return mode
	}
	return ""
}
```

Empty when the SDK omits them. The PRD `host` / `auction_owner` / `integration_mode` model is not implemented.

---

## 2. Spike answers (backend)

Numbering matches the Linear initiative.

### Q1. Does Bidon emit telemetry today?

**Yes, on the server.** The SDK already posts domain events (`/v2/stats`, `/v2/show`, …) and the server additionally emits auction/DSP/notice events to Kafka. This is **not** the PRD schema.

**Cutover:** dual-emit. Old topics stay until the unnamed downstream consumer of `ad-events` / `notification-events` is migrated. A hard cutover is unsafe.

Who consumes those topics is **not in this repo** (README mentions a `bidon-ad-events` topic historically; compose uses `ad-events`). Owner must be named before monthly cost can close. See §6 below.

### Q4. Direct-mode auction mechanics

- **Rounds:** one server bidding round. `bidding.Builder.HoldAuction` fans out to adapters in parallel (`internal/bidding/builder.go`). `AuctionResult.RoundNumber` is hardcoded to `0`.
- **Price floor:** per-request. Mixed from SDK `auction_pricefloor`, auction-config floor, cached ads, and (custom-adapter) `previous_auction_price` (`auction.priceFloor`).
- **Waterfall:** server returns ranked `ad_units` + `no_bids`. The SDK walks the waterfall. Re-sort after a failed load is client-side. Server sees the outcome later on `/v2/stats`.

### Q5. Can the waterfall expose `waterfall_position` and `attempts_made`?

Not as first-class server fields today.

`/v2/stats` already receives ordered `ad_units[]` with per-unit `status`, `fill_start_ts` / `fill_finish_ts`, and `error_message` (`internal/sdkapi/schema/stats_request.go`). If the SDK preserves **attempt order**, ingest can derive:

- `waterfall_position` = index in `ad_units` (1-based)
- `attempts_made` = count of units with a terminal status, or `len(ad_units)`

If the array is not attempt-ordered (e.g. ranked or winner-first), that is an **SDK stats-payload** change, not a server waterfall refactor. M1 headline metrics (fill rate, time to fill, per-source load rate) are therefore **not blocked on a backend restructure**; they are blocked on the SDK preserving order (or sending the two fields explicitly).

Listed separately in the ticket list because that confirmation is the largest remaining uncertainty on the stats path.

### Q9. Can `ad_impression` ride the ad-serving channel?

**Yes. It already does.**

`POST /v2/show/{ad_type}` is the impression callback. `notification.Handler.HandleShow` fires BURL (billing) from that request for bidding ads. The Kafka `show` event is best-effort *after* the HTTP handler returns.

Billing is coupled to `/show` succeeding, not to Kafka. The PRD exception is: keep billing on `/show` (or a successor ad-path call). Do **not** move it onto the SDK batch queue.

Warehouse `ad_impression` should be emitted from `/show` onto the new topic, with `event_id` dedupe if the SDK also sends `ad_impression` for reconciliation. Server copy from `/show` is source of truth for `billing_notice_sent`.

Gap: `EventSender` does not record HTTP status, latency, or retry count — and does not even *observe* the HTTP status (see §1.2). `billing_notice_sent` cannot satisfy the PRD field list, and delivery rate cannot be computed at all, until BE-M1-3.

### Q10. Reusable network client for telemetry ingest?

The bidding HTTP client (`otelhttp` transport, 4s timeout in `cmd/bidon-sdkapi/main.go`) is **outbound to DSPs**, not an ingest server.

- Server-emitted events can keep using `event.Logger` → Kafka.
- Client batch ingest is **new work** (endpoint, auth, dedupe, geocode). SDK retry/cert handling is SDK M0 (BID-47).

### Auction ID (PRD Note 1 vs code)

PRD: in M1/M2 the server mints `auction_id` and returns it in the auction response.

Code: `auction_id` is **required on the request** (`schema.AdObject`). `buildResponse` echoes `req.AdObject.AuctionID`. Server does not mint it.

Changing this is a coordinated SDK + OpenAPI change, not a silent backend fix. Deferred with owner: SDK + backend jointly. Until then, correlation uses the SDK-supplied id.

### OTel today

- Metrics: Echo Prometheus middleware, Redis cache observers, gRPC interceptors, `otelhttp` on the bidding client.
- Traces: `ConfigureOTel()` installs a Sentry span processor (`config/otel.go`). There is **no** auction-scoped trace with a child span per DSP.
- Sentry traces are not a funnel store and not the grain-B backend. Auction trees **will** export to **VictoriaTraces**; Sentry stays exceptions only.

### Remote config today

`POST /v2/config` returns `init.tmax`, adapter init configs, segment, `bidding.token_timeout_ms`. No sampling rates, no telemetry kill switch (BID-47).

---

## 3. PRD server events vs current names

| PRD event | Closest current event | Gap |
| --- | --- | --- |
| `auction_request_received` | `auction_request` | No `participant_count`, no DSP list; logged after build, not at request accept |
| `dsp_request_sent` | *(none)* | `bid_request` is post-hoc after the HTTP call. `StartTS` on `DemandResponse` is the **auction** start (`params.StartTS`), not send time |
| `dsp_response_received` | `bid_request` + `bid` | `Status` is an HTTP code string, not `bid/nobid/timeout/http_error/malformed`. No `nbr`, `creative_id`, `crtype`, `supports_omid`. `ParseBids` errors fold into `Error` |
| `dsp_response_rejected` | part of `bid_request` / immediate LURL | Below-floor bids send LURL in `HandleBiddingRound` but are not a distinct telemetry event. No `reason` enum (`below_floor/schema_invalid/missing_adm_fields/blocked_category/expired`) |
| `auction_completed` | *(none on server)* | Closest is client `stats_request`. Server could emit at bidding-round end (winner among DSP bids + network line items), which is **not** the same as fill |
| `win_notice_sent` | `NURL` | Missing success, http_status, retry_count, latency_ms |
| `billing_notice_sent` | `BURL` | Same |
| `loss_notice_sent` | `LURL` | Same; `TimeoutURL` is an extra current type |
| `notice_delivery_failed` | `Error` field on NotificationEvent | Not a distinct event; retry count not recorded |

Client-side PRD events (`sdk_init_*`, `adapter_token_*`, `ad_load_*`, `ad_impression` from the renderer, waterfall attempt events) are **not** emitted by this repo. They will arrive via the new ingest (or continue to be implied by domain endpoints during dual-emission).

M1 funnel metrics that need **both** sides:

| Metric | Server contribution | Still needs SDK |
| --- | --- | --- |
| Bid rate / DSP timeout rate | `dsp_response_received.outcome` | No |
| Notice delivery rate | upgraded notice events | No |
| Fill rate, time to fill, per-source load rate | `/v2/stats` order/fields | `waterfall_position` / `attempts_made` (or guaranteed attempt order) |
| Render rate | `/v2/show` as `ad_impression` | Renderer-side `ad_filled` if `/show` is not the fill signal |

---

## 4. Dual-emission rules

1. Keep writing `ad-events` and `notification-events` unchanged until the downstream owner signs off.
2. New PRD events go to a new topic (`telemetry-events`; the [events TRD](./TRD_BidOn_Telemetry.md) sink reads only this topic).
3. Do not widen `AdEvent` in place to look like the M0 envelope. That mixes PII policy and schema versioning.
4. `/v2/show` remains the billing trigger. New `ad_impression` / `billing_notice_sent` are additional writes from that handler, not a replacement of BURL.
5. If the SDK later posts `ad_impression` to ingest as well, dedupe on `event_id` (or a derived key `show:{auction_id}:{demand_id}` during transition).
6. Cutover of the old topics is a separate ticket after warehouse queries are rewritten. Not part of M1.

---

## 5. Proposed backend architecture (not implemented)

Three grains, three stores. Details in the store docs; this is the wiring.

```
SDK domain routes + POST /v2/telemetry
        │
        v
  bidon-sdkapi ── JSON envelope ──► Redpanda telemetry-events ──► Parquet sink ──► S3 ──► DuckDB
        │                              │                              (Iceberg later; CH later on same files)
        │                              └── dual-emit unchanged ──► ad-events / notification-events
        │
        ├── GET /metrics ── scrape ──► VictoriaMetrics ◄── vmalert ──► Alertmanager
        │
        └── OTLP ──► otelcol-contrib ──► VictoriaTraces          (GET trace_id)
                         └── spanmetrics ──► VictoriaMetrics     (p95 / paging; no auction_id labels)

/v2/show: BURL + ad_impression produce; does not query any of the above.
```

| Grain | Job | Store |
| --- | --- | --- |
| A | Catalog SQL (`sampling_rate`, join `auction_id`) | Redpanda `telemetry-events` → Parquet/Iceberg → DuckDB |
| B | Per-auction span tree | OTLP → **VictoriaTraces** |
| C | “Page if X is bad for N minutes” | **VictoriaMetrics** (scrape `/metrics` + spanmetrics) |

Keep Redpanda as the ingest bus. Add `telemetry-events` with the M0 envelope.

Two producers, one schema:

1. **Server-native** — `auction.Service` and `notification.EventSender`, fire-and-forget (same pattern as `EventLogger.Log` today). Must not delay the auction HTTP response.
2. **Client batch ingest** — new `POST /v2/telemetry`. Auth = existing app key + bundle. Server fills `country` from MaxMind. Dedupe on `event_id`.

`/v2/config` grows a `telemetry` object: per-event sampling rates, publisher kill switch. Per-auction kill switch on the auction response can wait; not required for M0.

OTel: span in `auction.Service.Run`, child span per DSP in `bidding.Builder`, child spans on notice sends. Export via **otelcol-contrib → VictoriaTraces**. Turn Sentry span export **off** the auction path; keep Sentry for exceptions. Spanmetrics from the collector into **VictoriaMetrics** — do not put `auction_id` on series.

Infra not in the Go ticket list: Compose/Coolify for VM, VT, otelcol, Parquet sink, object-storage bucket. Owners still unnamed.

Events warehouse (Parquet sink, Spaces, DuckDB): [TRD_BidOn_Telemetry.md](./TRD_BidOn_Telemetry.md). Ingest, sampling, dual-emit, and OTel stay in this spike.

---

## 6. Open items this repo cannot close

| Item | Why | Owner |
| --- | --- | --- |
| Who consumes `ad-events` / `notification-events` today | No consumer in this repo | Data / infra (unnamed) |
| Object-storage bucket, Parquet sink, VM, VT, otelcol | Stores are **chosen**; who runs them and monthly $ (volume unmeasured) are not | Infra (unnamed) |
| Server-minted `auction_id` | Breaking vs current required request field | SDK + backend jointly |
| Stats array = attempt order? | Determines whether Q5 is ingest-only | SDK |
| Impression definition (`ad_impression` vs OM viewable) | PRD assigns this to the OM spike | OM spike |
| AppLovin timeout / configured CPM / signal timeout (Q14–16) | MAX adapter, not this repo | Demand / SDK |
| BID-46 envelope sign-off | Schema freeze before backend types | Tiziano, Jonathan (per BID-46) |
| Ad-path latency gate (PRD target N/A) | Product decision | Initiative owner |

---

## 7. M0 + M1 backend tickets

Days are **TBC**. Tickets assume BID-46 envelope is signed off before M0 types land. They do **not** include Grafana dashboards, SDK work, Compose/Coolify for VM/VT/otelcol/sink, or Linear filing.

Waterfall/stats work is listed separately (highest remaining uncertainty on the client-reported path). DSP fan-out work is the highest uncertainty on the server-native path.

### M0 — foundation

| ID | Ticket | Days | Notes |
| --- | --- | --- | --- |
| BE-M0-1 | Shared envelope types + `schema_version` in a new `internal/telemetry` package (do not extend `AdEvent`) | TBC | Blocked on BID-46. Include `event_id`, host/auction_owner/integration_mode, sampling_rate |
| BE-M0-2 | `telemetry-events` Kafka topic + logger engine next to `internal/sdkapi/event` | TBC | Env var `KAFKA_TELEMETRY_EVENTS_TOPIC`; compose/dev/staging wiring. **Create the topic explicitly** — `config/kafka.go:30` has `AllowAutoTopicCreation`, so auto-creation would give one partition. Set partitions, retention, and **zstd** compression at creation |
| BE-M0-3 | `POST /v2/telemetry` ingest: batch parse, app-key auth, allow-list validation, `event_id` dedupe (client events only, `(app_id, event_id)`), MaxMind `country` | TBC | Requirements G3/G5/G8/G11. Never sample errors; never reject on `schema_version`; fail open if Redis is down |
| BE-M0-4 | `/v2/config` `telemetry` block: single `auction_sample_rate`, publisher kill switch | TBC | Storage: app settings or dedicated config table. Default `1.0` at current scale. Per-auction kill switch **not** in this ticket |
| BE-M0-5 | Privacy enforcement on the new stream: no IP/IFA/IDFV/city; COPPA reduced field set; scrub `error_message`; do not copy `raw_request`/`raw_response` | TBC | Old topics stay as they are during dual-emission |
| BE-M0-6 | OTel auction trace skeleton: span in `Service.Run`, child per DSP, exemplar-friendly | TBC | Export **otelcol → VictoriaTraces**. Sentry exceptions only. Spanmetrics → VictoriaMetrics (no `auction_id` labels). No new business events in this ticket |

### M1 — direct mode, server events

| ID | Ticket | Days | Notes |
| --- | --- | --- | --- |
| BE-M1-1 | DSP fan-out events: `dsp_request_sent` at send time; `dsp_response_received` with outcome enum; `dsp_response_rejected` with reason | TBC | Highest server uncertainty: map HTTP/timeout/parse/below-floor onto PRD outcomes; capture `nbr`/`creative_id`/`crtype`/`supports_omid` where adapters already parse them (many do not) |
| BE-M1-2 | `auction_request_received` at request accept + `auction_completed` at bidding-round end | TBC | Distinct from fill. Include winner_demand_source, clearing_price, participant_count, total_latency_ms |
| BE-M1-3 | Notice events: success, http_status, retry_count, latency_ms on win/billing/loss; add `notice_delivery_failed` | TBC | **Sequence first in M1.** `EventSender` currently returns `nil` for any HTTP response (§1.2), so notice delivery rate is not merely unreported — it is unobserved. Unblocks J4 / Goal 3. Retry on 5xx while here; consider the float-equality bid match in `handler.go:234` as a companion fix |
| BE-M1-4 | `/v2/show` emits `ad_impression` + `billing_notice_sent` onto `telemetry-events`; keep BURL behaviour | TBC | Delivery guarantee = the `/show` HTTP call. Dedupe: client `event_id`, or `show:{auction_id}:{demand_id}:{ad_unit_uid}` in transition |
| BE-M1-5 | Dual-write: existing `AdEvent`/`NotificationEvent` unchanged alongside new events | TBC | Feature-flag off-switch for the new topic only |
| BE-M1-6 | **Waterfall / fill (listed separately):** derive `waterfall_position` / `attempts_made` from `/v2/stats` **or** accept explicit SDK fields; emit data needed for fill rate / time to fill / per-source load rate | TBC | Confirm with SDK before scheduling (stats order vs payload change). Not a server waterfall rewrite |

### Suggested sequence

**Server-side first.** The PRD's strategic drivers are certification reconciliation (Goal 3) and per-DSP delivery evidence (Goal 4). Both are server-owned and have **zero SDK dependency**. The client ingest endpoint and sampling config are blocked on BID-46 and produce nothing until an SDK ships and publishers upgrade — which PRD rationale 6 says takes quarters.

Ordering M0 by ticket number would front-load BE-M0-3 and BE-M0-4 — the SDK-coupled work — and leave the business-case work until last. The phases below deliberately do not.

**Phase A — foundation (no SDK dependency)**

1. BE-M0-1 envelope types, BE-M0-2 topic + logger engine
2. BE-M0-5 privacy enforcement (before anything is written)
3. BE-M1-5 dual-write flag plumbing

**Phase B — server events (still no SDK dependency; this is the demo)**

4. BE-M1-3 **notice outcomes** — `EventSender` observes the HTTP response. Unblocks J4 and Goal 3, and is the smallest ticket with the largest business payoff
5. BE-M1-1 DSP fan-out outcomes, BE-M1-2 `auction_request_received` / `auction_completed` — instrument the auction path once
6. BE-M1-4 `/v2/show` emits `ad_impression` + `billing_notice_sent`
7. BE-M0-6 OTel spans (parallelisable with 5–6)

At the end of Phase B, with Go changes only, Bidon can answer: per-DSP bid / nobid / timeout rate, notice delivery rate, clearing price vs floor, and where the auction latency budget goes. No client transport, no ingest endpoint, no publisher upgrade cycle.

**Phase C — client ingest (gated on BID-46 / BID-47)**

8. BE-M0-3 `POST /v2/telemetry`, BE-M0-4 `/v2/config` telemetry block
9. BE-M1-6 waterfall / fill, only after SDK confirms stats ordering

Do not emit M1 events before BE-M0-1/2 exist, or dual-emission will invent a third schema. Phases B and C are otherwise independent and can run in parallel if there is capacity.

### Explicitly not ticketed here

- M2 `adapter_load_budget_exceeded` / `configured_cpm`
- M3 bid-request handling / inbound MAX notices
- Compose/Coolify: VictoriaMetrics, VictoriaTraces, otelcol-contrib, Parquet sink, bucket
- Iceberg catalog, ClickHouse-on-the-lake (events Phase 4), Grafana dashboards, PagerDuty
- Filing these rows in Linear
