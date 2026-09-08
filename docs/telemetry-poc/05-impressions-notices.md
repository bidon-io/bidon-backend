# 5 — Impression and notice outcomes

## Linear issue

**Title:** `POC: ad_impression and notice HTTP outcomes`

**User story.** As someone reconciling delivery, I want to know whether an auction was shown and whether the DSP accepted the billing/win/loss notice — including HTTP failures — on the same `auction_id` as the bidding-round events.

**Goal.** `/v2/show` emits `ad_impression` on `telemetry-events`. Notice sends record real HTTP status, retries, and latency, plus a distinct failure event. RisingWave funnel joins through impression and billing. Old `notification-events` stay as they are.

**Definition of done.**

- HTTP 500 in a unit test → `notice_delivery_failed` and `success=false`; HTTP 200 → `*_notice_sent` with `http_status=200`.
- Show handler → `ad_impression` produced; BURL still fired.
- `notification-events` JSON shape unchanged.
- After a show on the compose stack: `funnel_5m` impression/billing counts move; `notice_delivery_5m` reflects the HTTP class.

**Out of scope.** Float-equality bid match in `notification/handler.go`, client-side impression dedupe, `/v2/stats` waterfall, reject-reason taxonomy.

---

## MR instructions

Blocked on issue 2 (logger + envelope on the path). Extends issue 4 SQL.

### Fix `EventSender` observation

`internal/notification/event_sender.go` today:

```go
httpResp, err := es.HttpClient.Get(u.String())
// ...
return nil  // ANY HTTP response
```

Change the retry closure to:

- Return `err` on transport failure (keep `backoff`, max 3).
- Treat `status >= 500` as retryable (return an error wrapping the status).
- Treat `status >= 400 && status < 500` as terminal failure (do not retry; record status).
- `2xx` → success.

After retries, record `http_status` (last response if any), `retry_count`, `latency_ms` (whole send including retries), `success`.

Still write `event.NewNotificationEvent` to `notification-events` with the existing fields. You may put a short error string in `NotificationEvent.Error` as today; do not change the JSON field set in a breaking way.

### New telemetry events

Add names on the envelope constants:

```go
EventAdImpression          = "ad_impression"
EventWinNoticeSent         = "win_notice_sent"
EventLossNoticeSent        = "loss_notice_sent"
EventBillingNoticeSent     = "billing_notice_sent"
EventNoticeDeliveryFailed  = "notice_delivery_failed"
```

Map `Params.NotificationType`:

| Current `NotificationType` | Success event |
| --- | --- |
| `NURL` | `win_notice_sent` |
| `LURL` | `loss_notice_sent` |
| `BURL` | `billing_notice_sent` |
| `TimeoutURL` | `loss_notice_sent` (or keep name `timeout_notice_sent` if you prefer one extra constant — pick one and use it in SQL) |

On failure after retries: emit `notice_delivery_failed` **instead of or in addition to** the `*_sent` event. Prefer **both**: `*_sent` with `success=false` *or* only `notice_delivery_failed` with `dsp`, notice kind in `error_code` / a new `notice_type` field. RisingWave issue 4 SQL should use one convention — document it in `risingwave-init.sql` comments and follow it here.

Add to `Record` if missing: `Success *bool` / `success`, `RetryCount`, `NoticeType`.

`EventSender` needs `Telemetry *telemetry.Logger` and enough envelope fields (`AppID` is not on `Params` today). Add `AppID`, `SessionID`, `Country`, `AdFormat` to `notification.Params` **or** look up only what you have (`Bundle`, `AuctionID`, `AdType`, `DemandID`). Minimum for joins: `app_id` + `auction_id`. Thread `AppID` from callers (`HandleShow`, `HandleBiddingRound`, stats/win/loss handlers). If a caller cannot supply `AppID` without a wide refactor, put `0` and log once — but `/v2/show` **must** pass the real app id (`req` has it via `BaseHandler`).

### `/v2/show`

`internal/sdkapi/v2/apihandlers/show_handler.go`: after a successful resolve, emit `ad_impression` (envelope from request + `dsp` = `bid.DemandID`, `price` = `bid.GetPrice()`). Keep `EventLogger.Log` of the old `show` `AdEvent` and `HandleShow` (BURL).

`ShowHandler` gets `Telemetry *telemetry.Logger`. Wire in `v2.Router` / `main.go`.

Do not move billing onto the telemetry logger. BURL stays in `HandleShow`.

### Metrics

```go
NoticeDeliveryTotal = CounterVec{"type", "result"} // type=nurl|lurl|burl|timeout; result=2xx|4xx|5xx|timeout
AdImpressionTotal   = Counter
```

### RisingWave (`docker/telemetry/risingwave-init.sql`)

Add columns to the source if you added JSON fields (`success`, `retry_count`, `notice_type`). Recreate source/views if standalone cannot `ALTER` easily (POC: drop + create in init is OK).

Extend `funnel_5m` with left joins:

- `ad_impression` → `impressions`
- `billing_notice_sent` with success true (or `http_status` 200–299) → `billing_ok`

Add:

```sql
CREATE MATERIALIZED VIEW IF NOT EXISTS notice_delivery_5m AS
SELECT window_start, notice_type, success, COUNT(*) AS n
...
```

Use whatever field you chose for notice kind.

### Tests

- `event_sender` test with `httptest.Server` returning 500 then 200 (retry) and 500×4 (failed). Assert telemetry mock + old `NotificationEvent` still produced.
- `show_handler_test.go`: logger present → one `ad_impression`; `HandleShow` still invoked (existing mock).

### Do not

- Change `bid.Price == impression.GetPrice()` matching (note it in the MR description as known debt).
- Add `/v2/stats` waterfall events.
