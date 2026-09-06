# Onboard a new network

Canonical registration lives in the Go network registry. Admin UI network lists and account-type dropdowns come from `GET /api/networks`.

## Checklist

1. **Adapter key** — add `Key` constant (and include it in `Keys` if still maintained) in `internal/adapter/adapter.go`.
2. **Network registry** — add a `Network` entry in `internal/adapter/network_registry.go`:
   - `Key`, `Label`, `AccountType` (`DemandSourceAccount::<Name>`)
   - `SupportsBidding` / `SupportsWaterfall`
   - Field projections: `AccountExtra` / `AppData` / `AdUnitExtra` via `CopyKey` (same name) or `RenameKey`
   - Optional `InjectEnvSecrets` when demand secrets come from env (Meta, Moloco pattern)
3. **OpenAPI adapter key** — add the key to `internal/admin/openapi/schemas/adapter-key.schema.json` (and regenerate admin API if needed).
4. **RTB adapter** (bidding networks) — implement under `internal/bidding/adapters/` and register the builder in `biddingAdapters` (`internal/bidding/adapters_builder/adapters_builder.go`). Proprietary paths (e.g. Amazon `FetchBids`) stay explicit beside the registry.
5. **SDK init config** — add a case / struct in `internal/sdkapi/adapter_init_config.go` (and any store special cases).
6. **Admin extras / validation** — line-item and demand-source account OpenAPI extra schemas + admin validators for network-specific fields; UI Yup extras for forms when the network has custom fields.
7. **Seeds** — sample demand source, accounts, app demand profiles, auctions, and line items as needed.

Runtime config remaps for ordinary networks are applied from the registry by `AdaptersConfigBuilder` — no per-key switch entry for standard account/app/ad-unit maps.

## Design questions for a new integration

- How is the creative delivered (proxy vs DSP URL)?
- How do SDK request ad types map to ad formats?
- What belongs in the Bidon rendering config (close / container / endcard / type)?
- Is there a protobuf (or other) bid-request contract beyond OpenRTB?
