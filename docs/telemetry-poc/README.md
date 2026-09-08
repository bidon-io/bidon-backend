# Telemetry POC — Linear issues and MR briefs

Six Linear issues, five MRs. File the parent as the project/epic; file `01`–`05` as issues. Each `0N-*.md` is split:

1. **Linear issue** — user story, goal, definition of done. Paste into Linear. No implementation.
2. **MR instructions** — for the agent that implements that issue. Specific to this repo.

| # | Linear title | File | Blocks |
| --- | --- | --- | --- |
| 0 | Telemetry POC: server auction path | [00-parent.md](./00-parent.md) | — |
| 1 | Typed telemetry-events logger | [01-telemetry-logger.md](./01-telemetry-logger.md) | 0 |
| 2 | Auction and DSP events plus metrics | [02-auction-events-metrics.md](./02-auction-events-metrics.md) | 1 |
| 3 | Auction and DSP traces | [03-traces.md](./03-traces.md) | 1 |
| 4 | Local live funnel (RisingWave) and observe stack | [04-observe-stack.md](./04-observe-stack.md) | 1 |
| 5 | Impression and notice outcomes | [05-impressions-notices.md](./05-impressions-notices.md) | 2, 4 |

```
0 parent
  ├── 1  typed telemetry logger
  │     └── 2  auction events + metrics
  │           ├── 3  traces          (∥ 4)
  │           └── 4  RisingWave + VM/VT/Grafana
  │                 └── 5  show + notices
```

Labels: `telemetry`, `poc`. Team: backend.

**Not in this POC:** Parquet / S3 / DuckDB, protobuf + Schema Registry, `/v2/telemetry`, sampling, Coolify, product UI.
