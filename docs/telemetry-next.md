# Telemetry: what landed, and what to do next

Start here if you are picking up Bidon telemetry after the 2026 wrap-up.
This file is the status of [PR #963](https://github.com/bidon-io/bidon-backend/pull/963)
(`bac-73-sdk-telemetry-ingest`) against the BAC-91 tickets. The design notes
live beside it; this page says which of them the code actually implements.

The branch is rebased onto `new-main`, which already has the PRD envelope
([#962](https://github.com/bidon-io/bidon-backend/pull/962)) and Smadex
([#934](https://github.com/bidon-io/bidon-backend/pull/934)). Ingest keeps that
header. `app_key` and `bundle` are written on `envelope.app`, and
`schema_version` is the int32 `1`. A client `host` or `auction_owner` string
(`publisher`, `max_adapter`, `bidon`, `max`) is stored on
`envelope.integration` as the PRD enum.

## What this pull request implements

`POST /v2/telemetry` is served by `bidon-telemetry`, a Rust process on port
**1326**. sdkapi does not register the route (the contract is `telemetry/openapi.yaml`, outside the sdkapi spec). The handler allow-lists the client catalogue, stamps `app_key`,
`bundle`, and `schema_version`, and fail-open produces to `telemetry-events`.
An unknown event or metric is dropped. The batch still returns 202. A batch
that is too large returns 413.

| Ticket | In this PR |
| --- | --- |
| [BAC-74](https://linear.app/bidon/issue/BAC-74) client events catalogue | Yes. Named client messages in `schemas/proto/org/bidon/telemetry/v1/events.proto`, registered in `telemetry/src/catalog.rs`. |
| [BAC-75](https://linear.app/bidon/issue/BAC-75) client metrics catalogue | Yes. `MetricObservation` in the same proto. Unknown names and extra label keys are dropped, not a 400. |
| [BAC-76](https://linear.app/bidon/issue/BAC-76) batched ingest | Yes. `telemetry/` (Rust). Fixtures under `telemetry/testdata/`. Try them from `sdk.http` against `http://localhost:1326`. |
| [BAC-77](https://linear.app/bidon/issue/BAC-77) ingest observations on `/metrics` | Partial. `bidon-telemetry` exposes `ingest_total{result}` (`accepted`, `dropped_name`, `dropped_metric`, `dropped_invalid`, `buffer_full`, `produce_error`) and three client histograms on `:1326/metrics`. There is no `client_event_total{event_name}`. These series are on the Rust process, not on sdkapi `:1324/metrics`. VictoriaMetrics scrapes both (`docker/telemetry/victoriametrics/scrape.yml`). |
| [BAC-78](https://linear.app/bidon/issue/BAC-78) kill switch and allow-lists on `/v2/config` | Partial. `TELEMETRY_INGEST_ENABLED=false` makes ingest return `accepted: 0` and leaves auctions alone. The server does not re-sample. `/v2/config` does **not** return `telemetry.enabled`, `sample_rate`, `event_names`, or `metric_names`. The intended shape is under "BAC-78 config contract" below. |
| [BAC-79](https://linear.app/bidon/issue/BAC-79) OpenAPI for ingest and config telemetry | Partial. `telemetry/openapi.yaml` documents `POST /v2/telemetry` (202, 413, problem responses) with its own `servers:` entry. The config `telemetry.*` block is below and is not served yet. |
| [BAC-62](https://linear.app/bidon/issue/BAC-62) OpenTelemetry auction spans | No. See below. |

Closed siblings, already replaced by this PR: [#957](https://github.com/bidon-io/bidon-backend/pull/957)
(WIP OpenAPI on the old stacked history). There is no separate PR for
BAC-74, 75, 76, 77, 78, or 79. Those tickets were folded into this branch.

## Already on new-main

Do not re-implement these.

- Typed `telemetry-events` logger, auction and DSP events, sdkapi `/metrics` counters ([#954](https://github.com/bidon-io/bidon-backend/pull/954), BAC-60/61).
- PRD envelope header on auction and DSP events ([#962](https://github.com/bidon-io/bidon-backend/pull/962)).
- VictoriaMetrics scraping sdkapi, described in [telemetry-poc.md](./telemetry-poc.md).

## Parked outside this pull request

| Work | Where it is | What to do |
| --- | --- | --- |
| [BAC-62](https://linear.app/bidon/issue/BAC-62) auction parent span and one child per DSP | Branch `bac-62-poc-opentelemetry-auction-parent-and-per-dsp-child-spans`. No open PR. The branch still contains the pre-merge BAC-60/61 commits, so it is not a clean diff on `new-main`. Brief: [03-traces.md](./telemetry-poc/03-traces.md). | Rebase onto `new-main` before treating the diff as the patch. VictoriaTraces and otelcol are not in compose. `OTEL_EXPORTER_OTLP_ENDPOINT` unset must leave auctions unchanged. |
| [BAC-64](https://linear.app/bidon/issue/BAC-64) `ad_impression` and notice HTTP outcomes | Open [PR #961](https://github.com/bidon-io/bidon-backend/pull/961), one commit, not rebased onto the PRD envelope. Brief: [05-impressions-notices.md](./telemetry-poc/05-impressions-notices.md). | Rebase onto `new-main`. Show still fires BURL. `notification-events` JSON stays as it is. |
| Observe stack: RisingWave, VictoriaTraces, otelcol, Grafana ([BAC-63](https://linear.app/bidon/issue/BAC-63), BAC-85, BAC-86, BAC-87, BAC-88) | Not started. `docker/telemetry/risingwave-init.sql` is a draft and is not mounted by compose. Brief: [04-observe-stack.md](./telemetry-poc/04-observe-stack.md). How to read the bus and vmui once events exist: [explore-console-and-vmui.md](./telemetry-poc/explore-console-and-vmui.md). The screenshots in that note were taken on a stack that also produced show and notice rows, which this PR does not emit. | Leave parked unless someone is going to operate the stack. |
| Scalar OpenAPI docs ([BAC-80](https://linear.app/bidon/issue/BAC-80)) | Not started. | The spec in `openapi.yaml` is enough to generate a client. |
| Events warehouse | Design only. [TRD](./TRD_BidOn_Telemetry.md): Parquet on DigitalOcean Spaces, one table per event type, DuckDB for the SQL. No sink in this repo. | Build the sink only after `telemetry-events` on `new-main` is the schema you mean to keep. |

## Docs in the repo

Design and research, from the BAC-53 spike. They describe the intended system.
They are not a claim that this PR built it.

| Doc | What it is |
| --- | --- |
| [adr/0002-telemetry-ingest.md](./adr/0002-telemetry-ingest.md) | Why client ingest is a separate Rust process. |
| [telemetry-brief.md](./telemetry-brief.md) | Short brief. |
| [telemetry-requirements.md](./telemetry-requirements.md) | Jobs to be done, grains A/B/C. |
| [telemetry-m0-m1-backend-spike.md](./telemetry-m0-m1-backend-spike.md) | What sdkapi already emits, PII, dual-emit, ticket list. |
| [PRD v1](./PRD_BidOn_Telemetry%20-%20v1.pdf) | Product requirements. PDF only. |
| [TRD_BidOn_Telemetry.md](./TRD_BidOn_Telemetry.md) | Events warehouse (grain A). |
| [telemetry-events-store.md](./telemetry-events-store.md) | Why Parquet/Iceberg, not a database, for events. |
| [telemetry-traces-store.md](./telemetry-traces-store.md) | VictoriaTraces for grain B. |
| [telemetry-metrics-store.md](./telemetry-metrics-store.md) | VictoriaMetrics for grain C. |
| [telemetry-storage-sizing.md](./telemetry-storage-sizing.md) | Volume numbers. |
| [telemetry-storage-recommendation.md](./telemetry-storage-recommendation.md) | Unified ClickHouse, declined. |
| [telemetry-poc/](./telemetry-poc/README.md) | Original issue briefs (`00`–`05`). Historical. Status is this file. |
| [telemetry-poc.md](./telemetry-poc.md) | How to query VictoriaMetrics locally. |

## If you continue the work

1. Run the PR test plan: `go test ./internal/telemetry/ ./internal/sdkapi/v2/app/ ./internal/sdkapi/v2/api/`, `cargo test --manifest-path telemetry/Cargo.toml`, then `just compose` and POST `telemetry/testdata/valid_request.json` to `http://localhost:1326/v2/telemetry`. Confirm one framed record on `telemetry-events`, counters on `:1326/metrics`, and that `:1324` does not serve the route. POST `unrecognized_request.json` and confirm 202 with the unknown name dropped.
2. Finish BAC-78 on sdkapi: add `telemetry.enabled`, `sample_rate`, `event_names`, and `metric_names` to `/v2/config`, using the WIP yaml as the contract. Leave sampling on the SDK.
3. Rebase [#961](https://github.com/bidon-io/bidon-backend/pull/961) and the BAC-62 branch separately. Do not fold them into this PR.
4. Leave the warehouse, RisingWave, VictoriaTraces, and Grafana until someone is going to run them. The TRD and the store notes are the design.

## BAC-78 config contract

Additive `telemetry` object on the `POST /v2/config` response, served by
sdkapi. Sampling stays on the SDK; the server does not re-sample.

```yaml
telemetry:
  enabled: true          # false: SDK must not POST; ingest still 202s with accepted 0
  sample_rate: 0.1       # coherent per-auction keep probability, 0..1; funnel and errors stay at 1 on the SDK
  event_names:           # live allow-list, subset of the proto catalogue
    - ad_request_started
    - ad_filled
  metric_names:          # live allow-list, subset of the metric catalogue
    - client_auction_duration_seconds
    - client_token_collection_duration_seconds
    - client_ad_load_duration_seconds
```

Disabled looks like `enabled: false, sample_rate: 0, event_names: [], metric_names: []`.
