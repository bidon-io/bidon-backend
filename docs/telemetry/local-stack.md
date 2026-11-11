# Local telemetry stack: Redpanda Console and VictoriaMetrics

For the local `just compose` stack. Console is the event log: one record per thing that happened. VictoriaMetrics (vmui) is the counters and histograms scraped from the APIs. An auction id lives on the event. It is not a metric label.

| | URL |
| --- | --- |
| Redpanda Console | http://localhost:8080 |
| Schema Registry (also linked from Console) | http://localhost:18081 |
| vmui | http://localhost:8428/vmui |
| Scrape target health | http://localhost:8428/targets |
| sdkapi metrics, before they are scraped | http://localhost:1324/metrics |
| Telemetry ingest metrics | http://localhost:1326/metrics |

VictoriaMetrics scrapes both `/metrics` endpoints every 15 seconds and keeps 7 days (`docker/telemetry/victoriametrics/scrape.yml`). A counter moves on the raw `/metrics` page immediately. vmui shows it on the next scrape.

To put data on the bus, use `sdk.http`:

- `POST http://localhost:1324/v2/auction/{ad_type}` writes the server auction and DSP events.
- `POST http://localhost:1326/v2/telemetry` writes client events onto the same telemetry topic and updates the client histograms.

`just sdk-api` on the host, without `USE_KAFKA=true`, does not write the topic. The same server events are protojson lines in the sdkapi log.

## What you are looking at

Three topics. Only one of them is the new catalog.

| Topic | Who writes it | Payload | When it has rows |
| --- | --- | --- | --- |
| `telemetry-events` | sdkapi, and `bidon-telemetry` for client batches | Protobuf, one message type per event, Confluent-framed | After an auction or `POST /v2/telemetry` |
| `notification-events` | sdkapi, when it calls a DSP notice URL | JSON | After a win, loss, billing, or timeout notice |
| `ad-events` | sdkapi | JSON, the older auction log (includes device ids and raw bid bodies) | Off in local compose. `KAFKA_PRODUCE_AD_EVENTS` is `false`, so this topic is often missing |

`telemetry-events` is the topic to explore. The other two are the streams that already existed.

## Redpanda Console

Open **Topics**, then `telemetry-events`, then **Messages**. Set the start offset to the oldest offset (or a timestamp just before you sent the request) and raise the limit. A three-DSP auction with no show is eight records: one request, three sends, three responses, one completion. A below-floor bid adds a `dsp_response_rejected`.

Do not use **Produce record** or **Delete records** while poking around. Those write or wipe the topic.

Records are not keyed. The table timestamp is when the record was produced, not `eventTs`. **PROTOBUF · N B** in the value preview means Console decoded it; a binary blob means the Confluent frame was missing.

The **Schema** link on a record can name a different message (for example `SignalFailed` on an `ad_request_started` row). Every subject is the same `events.proto` file, so the registry stores them under one schema id and Console prints one subject name for it. The message index inside the record selects the type. Trust `envelope.eventName` or the `event_name` header.

Each record has two headers and a decoded value.

| Header | Example | Use it for |
| --- | --- | --- |
| `event_name` | `dsp_response_received` | Which catalog event this row is |
| `protobuf_message` | `org.bidon.telemetry.v1.DspResponseReceived` | Which protobuf type Console used to decode it |

Console prints protobuf as JSON with camelCase names, so the proto field `auction_id` appears as `auctionId`, and `event_name` inside the envelope appears as `eventName`. Enums appear as their proto names (`OUTCOME_BID`, `ERROR_CODE_NO_ADS_FOUND`), not the short metric labels.

Every event has an `envelope`:

| Field in the UI | Meaning |
| --- | --- |
| `eventId` | Unique id for this row |
| `eventName` | Same string as the `event_name` header |
| `eventTs` | Unix milliseconds |
| `schemaVersion` | `1` |
| `appId` | Server app id. Set on sdkapi events. Client ingest writes `0` and fills `appKey` and `bundle` instead. int64, so Console shows it as a string |
| `auctionId` | The id on the auction request. Join key, together with `appId` |
| `sessionId` | SDK session |
| `adType`, `adFormat`, `country` | From the auction request when sdkapi emitted the row. Client ingest leaves `country` empty |
| `dspId`, `demandSource` | Filled on DSP rows |
| `samplingRate` | `1`. Nothing is sampled yet |
| `traceId` | Reserved for the per-auction trace. Empty for now |
| `app.sdkVersion` | Client rows only: the `X-Bidon-Version` header |

The first records after sdkapi boots can be opaque bytes. Registration of each schema is async, and until that type has a schema id the producer sends raw protobuf. Those rows have the same headers, and Console cannot expand the value. Send the auction again, or read a later offset. Subjects show up under **Schema Registry**, named `telemetry-events-org.bidon.telemetry.v1.<Message>`.

### Filtering

On the Messages page, **Add filter → JavaScript Filter**. The function returns `true` to keep the record. Headers arrive as bytes, so decode them. This works even when the value is still raw protobuf.

```javascript
function header(name) {
  const raw = headers[name];
  if (!raw) return "";
  return String.fromCharCode(...new Uint8Array(raw));
}

return header("event_name") === "dsp_response_received";
```

One auction, all of its rows:

```javascript
function header(name) {
  const raw = headers[name];
  if (!raw) return "";
  return String.fromCharCode(...new Uint8Array(raw));
}

return value.envelope && value.envelope.auctionId === "PASTE_AUCTION_ID";
```

A DSP that timed out:

```javascript
return value.outcome === "OUTCOME_TIMEOUT";
```

A below-floor reject:

```javascript
return value.rejectReason === "REJECT_REASON_BELOW_FLOOR";
```

On `notification-events` the payload is already JSON, so the field names stay snake_case:

```javascript
return value.event_type === "NURL";
```

### Server events on `telemetry-events`

sdkapi emits these. `POST /v2/telemetry` refuses the five names in this table.

**Bidding round.** One auction emits the first, then one send and one response per DSP that was actually called, then the completion. `participantCount` on the completion equals that DSP count. Amazon is one response per request, not one per slot.

| `event_name` | When | Fields past the envelope |
| --- | --- | --- |
| `auction_request_received` | `/v2/auction` is accepted, before the effective floor replaces the SDK floor | `requestedPriceFloor` — the floor the SDK sent |
| `dsp_request_sent` | Immediately before the adapter HTTP call | `dsp`, `scope` (`SCOPE_BIDDING_ROUND`), `timeoutBudgetMs` (0 for Amazon) |
| `dsp_response_received` | Adapter returned, once per send | `dsp`, `outcome`, `httpStatus`, `latencyMs` (send to return), `price` when it bid, `nbr` when the response had one, `creativeId`, `crtype` |
| `dsp_response_rejected` | A bid existed and `price` did not beat the effective floor | `dsp`, `price`, `priceFloor`, `rejectReason` (`REJECT_REASON_BELOW_FLOOR` only, today) |
| `auction_completed` | Bidding round finished, including failures | `winnerDsp`, `price`, `priceFloor` (effective floor, not the SDK floor), `participantCount`, `totalLatencyMs`, `errorCode` |

`outcome` on the protobuf row:

| UI value | What happened |
| --- | --- |
| `OUTCOME_TIMEOUT` | The call hit its deadline |
| `OUTCOME_HTTP_ERROR` | HTTP 4xx/5xx, or a transport error with status 0 |
| `OUTCOME_MALFORMED` | A body came back and could not be parsed |
| `OUTCOME_BID` | A bid |
| `OUTCOME_NOBID` | A clean no-bid (204 or an empty seat) |

Timeout is chosen before the HTTP and parse checks. An adapter whose `ExecuteOptions` failed (missing credentials, bad adapter config) still has a `dsp_request_sent` and a response with `OUTCOME_HTTP_ERROR` and status 0, even though no request left sdkapi. `adapter_options_errors_total` tells the two apart (see vmui below). `errorCode` on `auction_completed` is `ERROR_CODE_NO_ADS_FOUND`, `ERROR_CODE_INVALID_AUCTION_KEY`, or `ERROR_CODE_ERROR`. Empty means the round finished without an auction error. `winnerDsp` is empty when nothing cleared the effective floor.

Show and notice events (`ad_impression`, `win_notice_sent`, `billing_notice_sent`, `loss_notice_sent`, `notice_delivery_failed`) and their metrics come with [#961](https://github.com/bidon-io/bidon-backend/pull/961) (BAC-64). They are not on `new-main` yet.

### Client events on the same topic

`POST /v2/telemetry` on port 1326 publishes every other catalog name. Filter with the same `event_name` header. The envelope on these rows has `appKey` and `bundle`, `appId` 0, and no `traceId`. An unknown or server event name, an event that does not decode as its message, and a metric that is not on the allow-list are dropped and counted on `ingest_total` (see vmui below). The HTTP response is still 202.

| Area | `event_name` |
| --- | --- |
| Init | `sdk_init_started`, `sdk_init_completed`, `sdk_init_failed`, `adapter_init_result`, `config_fetch_result` |
| Tokens | `adapter_token_requested`, `adapter_token_result`, `token_collection_completed` |
| Client auction | `ad_request_started`, `auction_requested`, `auction_response_received`, `ad_filled`, `auction_failed`, `auction_no_demand` |
| Load | `ad_load_requested`, `adm_parse_result`, `renderer_selected`, `ad_loaded`, `ad_load_failed`, `ad_expired` |
| Show | `ad_show_requested`, `ad_impression`, `ad_show_failed`, `ad_viewable`, `ad_clicked`, `ad_closed`, `ad_reward_granted` |
| Render | `webview_error`, `render_crashed`, `sdk_crashed` |
| Video | `video_start`, `video_q1`, `video_midpoint`, `video_q3`, `video_complete`, `video_skipped`, `video_error` |
| MAX adapter | `adapter_load_budget_exceeded`, `no_fill_returned_to_max`, `signal_requested`, `signal_provided`, `signal_failed`, `signal_timeout_suspected` |

`sdk.http` has a “one of each client event” request that posts the catalogue fixture. After it, filter `telemetry-events` on any name in the table.

### The older JSON topics

`notification-events` is one JSON object per notice HTTP call. `event_type` is `NURL`, `BURL`, `LURL`, or `TimeoutURL`. Useful fields: `auction_id`, `demand_id`, `ecpm`, `url`, `error`, `loss_reason`. This is the call sdkapi made.

`ad-events` uses `event_type` values such as `config`, `auction_request`, `bid_request`, `bid`, `stats_request`, `demand_request`, `client_bid`, `win`, `loss`, `show`, `click`, `reward`. Local compose does not produce it. Those rows carry advertising ids, IP, and raw DSP bodies, which is why the new topic exists beside them.

## VictoriaMetrics

Open http://localhost:8428/vmui. The **Query** tab takes PromQL. Set the time range to the last 15 or 30 minutes, run the query, and switch between the table and the graph. **Explore → Explore cardinality** lists metric names and which labels they have. http://localhost:8428/targets should show `bidon-sdkapi` and `bidon-telemetry` as up.

Two jobs:

| `job` | Process | What landed there |
| --- | --- | --- |
| `bidon-sdkapi` | sdkapi `:1324` | Auction and DSP metrics, plus ordinary HTTP metrics |
| `bidon-telemetry` | ingest `:1326` | Client ingest results and the three client histograms |

Metric labels are low-cardinality on purpose. There is no `auction_id`, `app_id`, or `event_id`. To see one auction, use Console. vmui answers “how many” and “how slow” across auctions.

Counters only go up. `increase(...[15m])` is the count over the window. `rate(...[5m])` is per second. A histogram query uses the `_bucket` series.

### Is the scrape alive?

```promql
up
```

```promql
sum by (url, code) (increase(sdkapi_requests_total[15m]))
```

`sdkapi_requests_total` and `sdkapi_request_duration_seconds` exist as soon as sdkapi has served traffic. The telemetry series below appear after the first event of that kind.

### Auctions

`result` is `ok` or `error`. `error` means `Run` returned an error (`no ads`, bad auction key, or anything else). A round that finishes with no winner and no error is still `ok`; the empty `winnerDsp` is on the event, not on this counter.

```promql
sum by (result) (increase(auction_completed_total[15m]))
```

### DSP outcomes

`outcome` is the short label: `bid`, `nobid`, `timeout`, `http_error`, `malformed`. That is the same decision as `OUTCOME_*` on `dsp_response_received`, without the prefix.

```promql
sum by (dsp, outcome) (increase(dsp_response_total[15m]))
```

Timeouts for one DSP, as a share of its responses:

```promql
sum by (dsp) (rate(dsp_response_total{outcome="timeout"}[5m]))
/
sum by (dsp) (rate(dsp_response_total[5m]))
```

### DSP latency

`dsp_request_duration_seconds` is send-to-return, labeled only by `dsp`. Buckets are 50ms, 100ms, 250ms, 500ms, 1s, 2s, 4s, 8s. A latency above 8s sits in the `+Inf` bucket.

```promql
histogram_quantile(0.95, sum by (le, dsp) (rate(dsp_request_duration_seconds_bucket[5m])))
```

Swap `0.95` for `0.50` for a median. The graph is empty until that DSP has a few samples inside the window.

### Dropped telemetry

This should stay flat. A non-zero `reason` means a catalog event never reached `telemetry-events`.

```promql
sum by (reason) (increase(telemetry_dropped_total[15m]))
```

| `reason` | Meaning |
| --- | --- |
| `invalid` | Missing event name or envelope |
| `marshal` | Protobuf encode failed |
| `buffer_full` | The producer buffer was full. Counted, not logged on every row |
| `produce_error` | The broker rejected the record |

### Adapter misconfiguration

```promql
sum by (adapter) (increase(adapter_options_errors_total[15m]))
```

No series when every adapter built its request options. A series names the adapter key whose `ExecuteOptions` failed. It keeps climbing on every auction that reaches that adapter, after the once-per-app error log goes quiet.

### Client ingest

These series are on `job="bidon-telemetry"`. They stay absent until something calls `POST /v2/telemetry`.

```promql
sum by (result) (increase(ingest_total[15m]))
```

| `result` | Meaning |
| --- | --- |
| `accepted` | Event produced, or a metric sample applied |
| `dropped_name` | Unknown `event_name`, including the five server auction and DSP names |
| `dropped_invalid` | Known name, but the event did not decode as its message (bad field type, missing id, non-UUID `session_id`) |
| `dropped_metric` | Metric name not allow-listed, not a histogram, an unexpected label key, a label value outside its set, or a negative value |
| `buffer_full` | Ingest queue was full |
| `produce_error` | Produce failed |

The only client histograms, and the only labels they accept:

| Metric | Labels |
| --- | --- |
| `client_auction_duration_seconds` | `ad_type` |
| `client_token_collection_duration_seconds` | `ad_type` |
| `client_ad_load_duration_seconds` | `ad_type`, `result` |

`ad_type` is `banner`, `interstitial`, or `rewarded`. `result` is `loaded` or `failed`.

```promql
histogram_quantile(0.95, sum by (le, ad_type) (rate(client_auction_duration_seconds_bucket[15m])))
```

```promql
histogram_quantile(0.95, sum by (le, ad_type, result) (rate(client_ad_load_duration_seconds_bucket[15m])))
```

Anything else in the batch’s `metrics` array increments `dropped_metric` and does not create a series.

## One auction, both UIs

1. Send **Auction banner Adikteev — iOS** from `sdk.http` (the `local` environment in `http-client.env.json` points `sdkapiHost` at http://localhost:1324).
2. In Console, open `telemetry-events` from the newest offset and read `envelope.auctionId` off `auction_request_received`.
3. Filter on that `auctionId`. You should see the request, a send and a response per DSP, any below-floor reject, and `auction_completed`. `latencyMs` on each response is that DSP only. `totalLatencyMs` on the completion is the whole round.
4. Wait for a scrape, then in vmui run `sum by (dsp, outcome) (increase(dsp_response_total[15m]))`. The outcomes should match the `outcome` fields you just read, summed across every auction in the window, not just this one.

If Console has the events and vmui does not, check http://localhost:8428/targets and http://localhost:1324/metrics. If `/metrics` has the counter and vmui does not, the scrape is behind or the time range does not cover it. If `/metrics` does not have it either, that code path did not run.

## Schema registry down

The registry is fail-open: auctions keep working and the producer sends raw protobuf until registration succeeds. To check it, keep Redpanda up and give sdkapi an unreachable registry URL. Stop the compose sdkapi so ports 1324 and 50051 are free, then run one with the override:

```shell
docker compose -f docker-compose.dev.yml stop bidon-sdkapi
docker compose -f docker-compose.dev.yml run --no-deps --service-ports --rm \
  -e SCHEMA_REGISTRY_URL=http://127.0.0.1:9 \
  bidon-sdkapi
```

- At startup, and again on each backoff retry, the log warns `schema registry register; producing raw protobuf until it succeeds`.
- Send an auction. `/v2/auction` returns as usual.
- The new `telemetry-events` values have no leading `0x00` Confluent magic byte, so Console shows them as opaque bytes. Headers are still there, so the `event_name` filter works.
- `run` does not register the `bidon-sdkapi` network alias, so http://localhost:8428/targets shows the sdkapi target down for the duration. Read http://localhost:1324/metrics directly instead.

Stop it with Ctrl-C, then bring the normal service back:

```shell
docker compose -f docker-compose.dev.yml up -d bidon-sdkapi
```
