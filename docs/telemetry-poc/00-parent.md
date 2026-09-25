# 0 — Telemetry POC: server auction path

Linear issue only (epic / parent). No MR.

## Linear issue

**Title:** `Telemetry POC: server auction path (events, metrics, traces, live funnel)`

**User story.** As a Bidon engineer (and later BD / support), I want to observe an auction — which DSPs bid, no-bid, or timed out, whether the ad was shown, and whether the billing notice was accepted.

**Goal.** One `POST /v2/auction` (and optionally `/v2/show`) proves three grains: catalog events on `telemetry-events`, low-cardinality series on `/metrics`, a per-auction trace with a child span per DSP.

**Definition of done.**

- Docker compose setup for development and staging environments (running on Coolify -> Digital Ocean).
- One auction writes `auction_request_received`, per-DSP send/receive, `auction_completed` to `telemetry-events`.
- Those rows are `SELECT`able in RisingWave (`funnel_5m`, `dsp_outcomes_5m`, `auctions_in_flight`) and on a Grafana board.
- DSP outcome counts and request duration appear on `GET /metrics` and in Grafana → VictoriaMetrics.
- The auction trace (parent + one child per DSP) is in VictoriaTraces; `trace_id` is on the event.
- Optional close-out (issue 5): same `auction_id` joins to impression and an **observed** notice HTTP status.
- When telemetry infra (RisingWave / VM / VT) is down: `/v2/auction` still succeeds; `ad-events` still written.

**Out of scope.** Lake, protobuf bus, client ingest, sampling, staging deploy, waterfall fill, bid-match-by-price fix, reporting product.

**Children.** Issues 1–5 in [README.md](./README.md).
