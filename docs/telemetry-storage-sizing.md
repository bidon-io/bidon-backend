# Telemetry storage sizing

Planning model for [events](./telemetry-events-store.md), [traces](./telemetry-traces-store.md), and [metrics](./telemetry-metrics-store.md). Replace inputs with measured direct-mode traffic before using these figures as a bill.

## Inputs

| Input | Default |
| --- | --- |
| Bidding DSPs per auction (`D`) | 6 |
| Waterfall load attempts until fill (`W`) | 4 |
| Loss notices (`L`) | `D − 1` = 5 |
| Keep `dsp_response_received` (success/nobid) | 0.1 |
| Keep `adapter_token_result` / successful `ad_load_*` | 0.2 |
| Keep successful traces | 0.01 |
| Fill rate | 0.4 at scale (1.0 for the 1-ad unit) |
| Event JSON | 1.2 KiB on Redpanda → **200 B** Parquet |
| Trace span on VictoriaTraces | **150 B** (~10× from 1.5 KiB OTLP) |
| Metric sample on VictoriaMetrics | **1 B** + 20% index + 25% merge headroom |
| Redis `tel:evt:{uuid}` | **200 B**, 72 h |
| Event retention | diagnostics **90 d**; impression / billing / error **400 d** |
| Trace retention | **14 d** |
| Metric retention | **90 d** @ 15 s scrape |

Errors, `ad_impression`, notices, `auction_completed`, and `dsp_response_rejected` are never sampled.

## 1 DAU, 1 auction, 1 impression

Stored events (keep rates already applied):

| Event | Stored |
| --- | --- |
| `sdk_init_*` | 2 |
| `adapter_token_result` | 0.2 × 6 = 1.2 |
| `ad_request_started` + `ad_filled` | 2 |
| `ad_load_*` | 0.2 × 4 = 0.8 |
| `auction_request_received` + `auction_completed` | 2 |
| `dsp_request_sent` | 6 |
| `dsp_response_received` | 0.1 × 6 = 0.6 |
| `dsp_response_rejected` | 1 |
| `ad_impression` + billing + win | 3 |
| `loss_notice_sent` | 5 |
| **Total stored** | **~24** (raw emit ~37) |

Unfilled auction (no show / billing / win): **~18** stored events.

| Signal | Write for this ad | Formula |
| --- | --- | --- |
| Events (Parquet) | **4.8 KiB** | 24 × 200 B |
| Events (Redpanda) | **29 KiB** | 24 × 1.2 KiB |
| Events (Redis 72 h) | **4.8 KiB** | 24 × 200 B |
| Traces (if kept) | **1.5 KiB** | (2 + 6 + 2) spans × 150 B |
| Traces (expected) | **15 B** | 0.01 × 1.5 KiB |
| Metrics | **0** | counters increment; no new series |

Of the 24 events: **8** are 400 d (fill, impression, notices, `auction_completed`, reject), **16** are 90 d.

Retained events if this same ad is served every day:

```
8 × 200 B × 400 d + 16 × 200 B × 90 d  =  0.9 MiB
```

## Daily volume

```
auctions_day      = DAU × auctions_per_DAU
impressions_day   = auctions_day × fill
unfilled_day      = auctions_day − impressions_day
events_day        = auctions_day × 18 + impressions_day × 6
parquet_day       = events_day × 200 B
traces_day        = auctions_day × 0.01 × 1.5 KiB
redis             = events_day × 200 B × 3 d
redpanda          = events_day × 1.2 KiB × topic_retention_days
```

Events on S3 (retention full):

```
s3_events  = impressions_day × 0.9 MiB
           + unfilled_day    × (2 × 200 B × 400 d + 16 × 200 B × 90 d)
           = impressions_day × 0.9 MiB
           + unfilled_day    × 448 KiB
```

Metrics (independent of DAU), 5,000 series:

```
5_000 × (86400 / 15) samples/day × 1 B × 90 d × 1.20 × 1.25  ≈  4 GiB
```

## Worked results

**100k DAU**, 4 auctions/DAU, fill 0.4 → 400k auctions, 160k impressions, 240k unfilled.

```
events_day   = 400k × 18 + 160k × 6     = 8.16M
parquet_day  = 8.16M × 200 B            = 1.5 GiB / day
s3_events    = 160k × 0.9 MiB + 240k × 448 KiB  ≈ 140 GiB + 105 GiB  ≈ 245 GiB
traces_day   = 400k × 0.01 × 1.5 KiB    = 5.9 MiB / day
traces_14d   = 5.9 MiB × 14             ≈ 83 MiB
redis        = 8.16M × 200 B × 3        ≈ 4.6 GiB
```

**1M DAU**, 8 auctions/DAU, fill 0.4 → 8M auctions, 3.2M impressions, 4.8M unfilled.

```
events_day   = 8M × 18 + 3.2M × 6       = 163.2M
parquet_day  = 163.2M × 200 B           = 30 GiB / day
s3_events    = 3.2M × 0.9 MiB + 4.8M × 448 KiB  ≈ 2.8 TiB + 2.0 TiB  ≈ 4.8 TiB
traces_day   = 8M × 0.01 × 1.5 KiB      = 117 MiB / day
traces_14d   = 117 MiB × 14             ≈ 1.6 GiB
redis        = 163.2M × 200 B × 3       ≈ 91 GiB
```

| | Events write / day | Events on S3 | Traces write / day | Traces on VT (14 d) | Metrics on VM (90 d) | Redis (72 h) |
| --- | --- | --- | --- | --- | --- | --- |
| 1 DAU, 1 filled ad/day | 4.8 KiB | 0.9 MiB | 15 B | — | **4 GiB** | 5 KiB |
| 100k DAU | **1.5 GiB** | **~245 GiB** | 6 MiB | **83 MiB** | **4 GiB** | **~5 GiB** |
| 1M DAU | **30 GiB** | **~4.8 TiB** | 117 MiB | **1.6 GiB** | **4 GiB** | **~91 GiB** |

`30 GiB / day` is new event Parquet. `4.8 TiB` is that write kept for 90/400 days. Traces at 100% keep: `8M × 1.5 KiB = 12 GiB / day`, **~170 GiB** on VT after 14 d.

## Aggregation windows

| Window | Store | What is stored |
| --- | --- | --- |
| 15 s scrape, 5 min alert | Metrics | 5k series, not per-ad rows |
| This auction | Traces | Sampled tree (~10 spans), 14 d |
| Funnel / fill / `1/sampling_rate` | Events | Raw rows on S3 |
| 90 d diagnostics | Events | Short partition |
| 400 d impression / billing | Events | Long partition |

Spanmetrics rolls traces into metrics at collect time. Funnels stay on raw event rows.

## Measure

1. `event_name` count per `auction_id` vs ~37 raw / ~24 stored.
2. Histograms of `D` and `W`.
3. Mean JSON bytes; Parquet bytes/row (replaces 1.2 KiB / 200 B).
4. VM series count — no `auction_id` label.
5. VT disk / trees kept (replaces 1.5 KiB).
