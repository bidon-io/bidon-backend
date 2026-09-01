# ADR-0001: DSP simulator

**Status:** Accepted
**Date:** 2026-09-01
**Deciders:** Tiziano Perrucci
**Linear:** TBD
**Supersedes / Superseded by:** —
**Flow diagram:** [0001-dsp-simulator-flow.puml](0001-dsp-simulator-flow.puml)

---

## Context

Bidon's demand adapters are tested in isolation through a `RoundTripper` stub
(see `internal/bidding/adapters/adikteev/adikteev_test.go:45`). Nothing in the
repo exercises the real HTTP path end to end: an actual OpenRTB bid request over
the wire, an actual bid response body, and the win / billing / loss notification
round trip that follows.

Three existing contracts define what a DSP simulator must do to be truthful:

1. **Bid responses** are parsed centrally at
   `internal/bidding/adapters/parse.go:47`: the first seat and the first bid
   are taken, `nurl` / `burl` / `lurl` / `ext.signaldata` are mapped, and HTTP
   204 means no bid. The simulator must emit a response this parser accepts.
2. **Notifications** are fire-and-forget **GET** requests with macros already
   substituted into query params (`internal/notification/event_sender.go:53`,
   macro list at `:89-102`). Substitution happens only when a macro is the
   *entire* value of a query param.
3. **"Configured auctions"** live in Postgres: `apps` (`package_name`,
   `platform_id`) → `auction_configurations` (`ad_type`, `pricefloor`,
   `bidding[]`, `ad_unit_ids[]`) → `line_items` (`is_bidding`, `format`) →
   `demand_sources` (`api_key`).

Two further observations shape the design:

- `imp.displaymanager` carries the adapter key verbatim in every adapter
  (`internal/bidding/adapters/adikteev/adikteev.go:113`,
  `internal/bidding/adapters/moloco/moloco.go:137`, …), and that string equals
  `demand_sources.api_key`. It is a reliable DSP discriminator.
- Adapter RTB endpoints are **hardcoded in Go** — one constant per adapter —
  with the single exception of Bidmachine, which reads `endpoint` from
  `demand_source_accounts.extra`
  (`internal/bidding/adapters/bidmachine/bidmachine.go:194-197`).

A corpus of real captured bid requests already exists at
`internal/sdkapi/v2/apihandlers/testdata/auction/adikteev/*_bidreq.json`.

---

## Decision

Build **`bidon-dspsim`**: a standalone OpenRTB DSP simulator in
`cmd/bidon-dspsim` + `internal/dspsim`, packaged as a Dockerfile target and a
`docker-compose.dev.yml` service on port 1325.

### Decisions

| #  | Decision | Rejected alternative |
| -- | -------- | -------------------- |
| D1 | Standalone binary, **zero changes to bidon**. Only reads the shared Postgres schema and reuses `config/` and `internal/db` helpers. | In-process test double per adapter; adding an endpoint-override hook to every adapter (useful, but a separate change). |
| D2 | Read auction configuration directly from Postgres, read-only, as a snapshot refreshed on a ticker (`DSPSIM_CATALOG_TTL`) behind an `atomic.Pointer`. A failed refresh keeps the previous snapshot. | Calling the admin API (couples the simulator to auth and admin uptime); a static config file (drifts from what bidon actually sees). |
| D3 | Match an inbound request on `app.bundle` plus the ad type / format inferred from the shape of `imp[0]` (`instl=1` → fullscreen, banner size → banner/mrec). No match → HTTP 204 with the reason in `X-Dspsim-Nobid-Reason`. | Requiring an explicit sim-side request mapping (duplicates the app's own configuration knob). |
| D4 | Creative library as JSON indexed **DSP → creative type → creative**, selected via `imp.displaymanager`, falling back to a `default` bucket when the DSP has no bucket or no eligible creative. `text/template` rendering, validated at load so a bad file fails startup, not a bid. Embedded default file; overridable via `DSPSIM_CREATIVES_FILE`. | Hardcoded creatives in Go; a single flat creative list without the DSP dimension. |
| D5 | Four creative types mirroring the `creativeType` values observed in captured Adikteev `burl`s: `static_banner` (plain `<a><img></a>`), `mraid_banner`, `mraid_interstitial`, `vast_video` (VAST 3.0). | MRAID-only (would not cover static banner traffic that real DSPs serve). |
| D6 | Notification URLs carry the bid id in the path and **every** macro `event_sender.go:89-102` substitutes, each as the whole value of its own param, so unresolved `${…}` is detectable per-param. | Minimal `${AUCTION_PRICE}`-only URLs (would not catch substitution regressions). |
| D7 | In-memory bid registry keyed by bid id, TTL + size-capped, append-only notification log; notifications always answer 200, including for unknown bid ids, which are kept in an orphan list. | Persisting to Postgres/Redis (the simulator is a test instrument, not a system of record). |
| D8 | Reuse `config.Echo()`, `config.UseCommonMiddleware`, `config.UseHealthCheckHandler`, `internal/db.Open` so logging, request ids, recover, body dump and health checks behave exactly like the real services. | Bare `net/http` server (different observability from production). |
| D9 | Soft demand cross-check (`displaymanager` ∈ the auction's configured `bidding[]` / line items): warn and still bid, recording `demand_configured: false` on the bid. `DSPSIM_STRICT_DEMAND=true` turns it into a 204. | Always rejecting unconfigured demand (a real DSP would not and could not know). |

### Components

| Component   | Responsibility |
| ----------- | -------------- |
| `catalog.go`   | Postgres → in-memory index of configured auctions, keyed by bundle; ticker refresh + `POST /debug/reload` |
| `matcher.go`   | Infer ad type/format from `imp[0]` shape; pair with a configured auction, or return a typed no-bid reason |
| `creatives.go` | JSON library loader/validator, DSP → type → creative index, weighted selection, `default` fallback |
| `bidder.go`    | OpenRTB `BidResponse` construction, random pricing above the floor, macro URLs, in-memory `BidRecord` assembly |
| `store.go`     | `map[bidID]*BidRecord` registry, TTL + size eviction, orphan notifications |
| `server.go`    | Echo routes: `/openrtb/bid`, `/notify/{win,loss,billing}/:bidID`, `/creative/*`, `/debug/*` |

### Routes

| Route | Purpose |
| ----- | ------- |
| `POST /openrtb/bid` (alias `POST /`) | Bid endpoint; 200 + bid response, or 204 + `X-Dspsim-Nobid-Reason` |
| `GET\|POST /notify/win/:bidID` · `/notify/loss/:bidID` · `/notify/billing/:bidID` | nurl / lurl / burl receivers |
| `GET /creative/impression/:bidID` · `/creative/click/:bidID` · `/creative/track/:bidID/:event` | creative-side (VAST/HTML) tracking |
| `GET /creative/asset/:name` | generated SVG images; stubbed `video/mp4` |
| `GET /debug/bids` (`?dsp=`) · `/debug/bids/:bidID` · `DELETE /debug/bids` | inspect / clear the bid registry and orphans |
| `GET /debug/creatives` · `/debug/creatives/:dsp` | creative library with per-creative serve counts |
| `GET /debug/catalog` · `POST /debug/reload` | inspect / reload the Postgres snapshot **and** the creative library |
| `GET /health_checks` | DB health via `config.UseHealthCheckHandler` |

### Configuration

All env vars are prefixed `DSPSIM_`, except the shared `DATABASE_URL` (opened
read-only). Defaults: `PORT=1325`, `PUBLIC_URL=http://localhost:$PORT`,
`SEAT=dspsim`, `CUR=USD`, `PRICE_MULT_MIN=1.5` / `PRICE_MULT_MAX=3.0`,
`FALLBACK_FLOOR=0.5`, `MAX_PRICE=25`, `NO_BID_RATE=0`, `LATENCY_MS=0`,
`CATALOG_TTL=60s`, `BID_TTL=1h`, `MAX_BIDS=10000`, `CREATIVES_FILE=` (embedded),
`STRICT_DEMAND=false`, `SEED=0` (time-based; a value makes selection and
pricing deterministic).

Query overrides on `/openrtb/bid` — `?dsp=`, `?adtype=`, `?creative=` — exist
to pin behaviour for manual testing (`dspsim.http`).

---

## Consequences

### Positive

- Real HTTP + OpenRTB coverage where previously there was only a `RoundTripper`
  stub: JSON round trips, 204 handling, and the nurl/burl/lurl round trip,
  including detection of unsubstituted macros and orphan notifications.
- The same Postgres data "configure, then see it bid" path that production uses
  — editing an auction configuration or seed data changes simulator behaviour
  on the next refresh, no restart.
- New creatives and new DSP buckets are JSON edits, not code; the library is
  hot-reloadable via `/debug/reload`.
- Deterministic replay under a fixed `DSPSIM_SEED`, and deterministic manual
  runs via query overrides.
- Tests replay the real captured Adikteev requests, and responses are
  round-tripped through the actual shared parser.

### Negative / accepted

- **Bidon cannot reach the simulator without an out-of-band redirect** (hosts
  entry, HTTP proxy, or Bidmachine `endpoint` override in the database). This is
  the direct cost of D1 and is accepted: the simulator is driven standalone via
  `dspsim.http`, and adapter endpoint overrides are a separate, deliberate
  change (see follow-ups).
- The simulator **couples to the DB schema**: changes to `line_items` /
  `auction_configurations` will need matching changes here. Accepted because
  schema changes are migrations reviewed in this repo.
- Interstitial vs rewarded is indistinguishable on the wire (`instl=1` for
  both); the matcher resolves it via the catalog and falls back to a preferred
  ordering, which is logged. The `?adtype=` override exists when it matters.
- All state is in memory and lost on restart — intentional for an instrument.

### Follow-ups

1. Optional `DSPSIM_ENDPOINT` / per-adapter env endpoint overrides so a live
   bidon auction can be pointed at the simulator — a separate ADR; D1 keeps
   this one independent.
2. A dedicated seed app + auction configurations for simulator testing, so
   fixture bundles and DB bundles coincide without editing sample seeds.
3. Optional RAOT proxying of `mraid.js` and true video assets if a downstream
   consumer needs to actually render/play the creatives.
