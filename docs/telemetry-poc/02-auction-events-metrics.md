# 2 — Auction and DSP events plus metrics

## Linear issue

**Title:** `POC: auction and DSP events plus /metrics counters`

**User story.** As a server engineer, I want each bidding round to record request, per-DSP send/receive (bid / no-bid / timeout), and completion, plus simple rates I can scrape, so I can see funnel and adapter health without an SDK change.

**Goal.** Dual-write the server bidding-round catalog to `telemetry-events` and increment low-cardinality Prometheus series. `ad-events` keep working exactly as today.

**Definition of done.**

- One 3-DSP auction (bid / nobid / timeout) emits 1 + 3 + 3 + 1 telemetry events when the flag is on.
- DSP latency is send→return, not auction-start→return.
- `GET /metrics` exposes `dsp_response_total{dsp,outcome}`, `dsp_request_duration_seconds{dsp}`, `auction_completed_total{result}`. No `auction_id` label.
- Flag off: auction HTTP unchanged; no new topic writes; `ad-events` still written.
- Outcome mapping covered by a table test.

**Out of scope.** Notices, `/v2/show`, OTel spans, per-adapter code, waterfall fill.

---

## MR instructions

Blocked on issue 1 (`internal/telemetry` Logger + `Record`).

### Call sites

**`auction.Service`** (`internal/auction/service.go`)

Add `Telemetry *telemetry.Logger` (nil-safe).

Today `Run` defers `logEvents` (old `AdEvent`s). Keep that. Also:

1. **At the top of `Run`** (after `req := params.Req` is usable), emit `auction_request_received` with envelope from `params.App.ID`, `req.AdObject.AuctionID`, `req.Session.ID`, `req.AdType`, `req.AdObject.Format()`, `params.Country`, `req.AdObject.PriceFloor`.
2. **When `Run` finishes** (success or the existing `defer` path), emit `auction_completed`:
   - `scope`: `"bidding_round"`
   - `winner_dsp` / `price` if any bid won the server round (same winner rule as `buildResponse` / highest valid bid — do not invent a new auction winner).
   - `participant_count`: `len(auctionResult.BiddingAuctionResult.Bids)` when present, else 0.
   - `total_latency_ms`: now − a `started` timestamp taken at the beginning of `Run`.
   - `error_code` if `err != nil` (short; do not put raw DSP bodies in it).

Use `telemetryLogger.Log` with the same `params.LogErr` style as `EventLogger`.

**`bidding.Builder.processAdapter`** (`internal/bidding/builder.go`)

Add `Telemetry *telemetry.Logger` on `Builder`. Wire it from `cmd/bidon-sdkapi/main.go` next to `NotificationHandler`.

Today `StartTS` is set to `params.StartTS` (auction start). Change:

```go
sendStart := time.Now().UnixMilli()
// emit dsp_request_sent HERE (before ExecuteRequest / Amazon FetchBids)
demandResponse := bidder.Adapter.ExecuteRequest(...)
demandResponse.StartTS = sendStart
demandResponse.EndTS = time.Now().UnixMilli()
```

Same for Amazon: set send/end **per** `DemandResponse`, not `params.StartTS`.

After the adapter returns (and after `ParseBids` when that runs), emit `dsp_response_received` with `outcome`, `http_status` (`demandResponse.Status`), `latency_ms` (`EndTS - StartTS`), `price` if `IsBid()`, `dsp` = `string(DemandID)`.

If the bid exists and `Price() < auctionRequest.AdObject.GetBidFloorForBidding()` (or the floor already used for LURL in `HandleBiddingRound`), also emit `dsp_response_rejected` with `reject_reason=below_floor`. Do not build a larger reject taxonomy.

Envelope on DSP events: same `app_id` / `auction_id` / `session_id` / country as the auction. `Builder` does not have `App.ID` today — add them to `BuildParams` (`internal/bidding/builder.go` `BuildParams`) from `auction.Builder` / whoever calls `HoldAuction`. Thread `AppID`, `Country`, `SessionID`, `AdType`, `AdFormat` through. Do not geocode again.

### Outcome mapping

Put this in `internal/telemetry` (e.g. `outcome.go`) so tests do not import bidding internals more than needed:

| Condition | `outcome` |
| --- | --- |
| `errors.Is(err, context.DeadlineExceeded)` | `timeout` |
| `IsBid()` | `bid` |
| `err != nil` and body/parse failure | `malformed` |
| `Status >= 400` or `Status == 0` with err | `http_error` |
| no bid, no error (204 / empty seat) | `nobid` |

Precedence: timeout → http_error (4xx/5xx) → malformed (parse err) → bid → nobid. Table-test these.

Constants:

```go
const (
    OutcomeBid       = "bid"
    OutcomeNoBid     = "nobid"
    OutcomeTimeout   = "timeout"
    OutcomeHTTPError = "http_error"
    OutcomeMalformed = "malformed"
)
```

### Metrics

`internal/telemetry/metrics.go` using `github.com/prometheus/client_golang/prometheus/promauto` on the **default** registerer (`echoprometheus.NewHandler()` serves `DefaultGatherer`):

```go
var (
    DSPResponseTotal = promauto.NewCounterVec(..., []string{"dsp", "outcome"})
    DSPRequestDuration = promauto.NewHistogramVec(..., []string{"dsp"})
    AuctionCompletedTotal = promauto.NewCounterVec(..., []string{"result"}) // ok|error
)
```

Histogram buckets: `0.05, 0.1, 0.25, 0.5, 1, 2, 4, 8` seconds (adapter timeout is 4s in `cmd/bidon-sdkapi/main.go`).

Increment in the same places you emit events. Labels: `dsp` = demand key string; **never** `auction_id`.

### Old path

Do not change `prepareAuctionRequestEvent` / `prepareBiddingEvents` JSON. `logEvents` stays.

### Tests

- New table test for `OutcomeFromDemand(...)`.
- Extend `internal/auction/service_test.go` and `internal/bidding/builder_test.go` (or telemetry tests with a mock `LoggerEngine`) so a 3-DSP fixture records 8 events and the three metric series move. If pulling Prometheus in auction tests is messy, test event payloads in-package and metrics in `internal/telemetry/metrics_test.go` via a helper `ObserveDSP(dsp, outcome, seconds)`.
- Existing auction event tests (`MockEventLogger` for `ad-events`) must still pass.

### Wire-up

`cmd/bidon-sdkapi/main.go`: pass `telemetryLogger` into `auction.Service` and `bidding.Builder`.
