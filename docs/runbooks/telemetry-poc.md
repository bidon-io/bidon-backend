# Telemetry POC: local metrics

`just compose` starts VictoriaMetrics, which scrapes `bidon-sdkapi:1324/metrics`
every 15s (config: `docker/telemetry/victoriametrics/scrape.yml`). Nothing in the
stack depends on it, so stopping it does not affect `/v2/auction`.

## Where to look

| What | URL |
|------|-----|
| Raw sdkapi metrics | `curl http://localhost:1324/metrics` |
| VictoriaMetrics UI (vmui) | http://localhost:8428/vmui |
| Scrape target health | http://localhost:8428/targets |

## Queries

Paste into vmui:

```promql
rate(dsp_response_total{outcome="timeout"}[5m])
histogram_quantile(0.95, sum by (le, dsp) (rate(dsp_request_duration_seconds_bucket[5m])))
increase(auction_completed_total[5m])
```

`dsp_response_total`, `dsp_request_duration_seconds` and `auction_completed_total`
only exist once the BAC-61 telemetry counters are on the branch you run, and only
after at least one auction (send one with `sdk.http`). `sdkapi_*` HTTP metrics are
always present.

## Not covered yet

RisingWave, VictoriaTraces, otelcol and Grafana are tracked separately under BAC-63.
