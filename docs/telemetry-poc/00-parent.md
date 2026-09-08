# 0 — Telemetry POC: server auction path

Linear issue only (epic / parent). No MR.

## Linear issue

**Title:** `Telemetry POC: server auction path (events, metrics, traces, live funnel)`

**User story.** As a Bidon engineer (and later BD / support), I want to watch a live server auction — which DSPs bid, no-bid, or timed out, whether the ad was shown, and whether the billing notice was accepted — without reading PII off `ad-events` or waiting for a warehouse.

**Why now.** The PRD needs delivery evidence. The long-term store (Connect → Parquet → DuckDB) is the right retain path and too much infra for this stage. RisingWave as a **pluggable consumer** of a new topic gives near-real-time funnel SQL. Metrics and traces sit beside it, not instead of it.

**Goal.** One local `POST /v2/auction` (and optionally `/v2/show`) proves three grains: catalog events on `telemetry-events` queryable in RisingWave, low-cardinality series on `/metrics`, a per-auction trace with a child span per DSP. Existing Kafka topics and the ad path stay unchanged.

**Definition of done.**

- Dev compose + `TELEMETRY_ENABLED=true` is the only setup. No Coolify, no object storage, no SDK release.
- One auction writes `auction_request_received`, per-DSP send/receive, `auction_completed` to `telemetry-events`.
- Those rows are `SELECT`able in RisingWave (`funnel_5m`, `dsp_outcomes_5m`, `auctions_in_flight`) and on a Grafana board.
- DSP outcome counts and request duration appear on `GET /metrics` and in Grafana → VictoriaMetrics.
- The auction trace (parent + one child per DSP) is in VictoriaTraces; `trace_id` is on the event.
- Optional close-out (issue 5): same `auction_id` joins to impression and an **observed** notice HTTP status.
- Flag off or RisingWave / VM / VT down: `/v2/auction` still succeeds; `ad-events` still written.
- New stream has no IFA, IP, city, or raw DSP payloads.

**Out of scope.** Lake, protobuf bus, client ingest, sampling, staging deploy, waterfall fill, bid-match-by-price fix, reporting product.

**Children.** Issues 1–5 in [README.md](./README.md). Close this parent as “bidding-round demo” without issue 5 if needed; say so in the comment.
