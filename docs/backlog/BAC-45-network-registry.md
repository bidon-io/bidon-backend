# Backlog: Single network registry across runtime, admin, and UI

**Status:** In Progress (inventory frozen)  
**Linear:** [BAC-45](https://linear.app/bidon/issue/BAC-45/single-network-registry-across-runtime-admin-and-ui)  
**Branch:** `feature/BAC-45/network-registry`  
**Base:** `new-main`  
**Related:** [BAC-21](https://linear.app/bidon/issue/BAC-21/improve-maintainability-of-adapters-in-sdk-api), [BAC-22](https://linear.app/bidon/issue/BAC-22/edit-adikteev-sdk-instance-id-in-admin-ui)

---

## Summary

Adding or changing a network today requires hand-updating scattered Go switches, admin validators, and UI constant maps. This work introduces one canonical **network/adapter registry** as the source of truth for runtime config wiring, admin catalog, and (eventually) UI network lists.

---

## Motivation

- Cut the “new network” tax described in [`docs/network.md`](../network.md).
- Stop maintaining parallel lists in Go (`adapter.Keys`, `biddingAdapters`, `AdaptersConfigBuilder` switch) and UI (`Networks.js`, `DemandSourceOptions.js`).
- Make account/app/ad-unit field remaps declarative instead of an unbounded per-key switch.
- Let the admin UI consume a catalog API instead of static lookup maps.

---

## Current inventory (registration surfaces)

| Surface | Location | What is duplicated |
|---------|----------|--------------------|
| Adapter keys | `internal/adapter/adapter.go` (`Key` consts + `Keys`) | Canonical runtime IDs |
| Bidding builders map | `internal/bidding/adapters_builder/adapters_builder.go` (`biddingAdapters`) | Which keys have RTB builders |
| Config field remaps | same file (`AdaptersConfigBuilder.Build` switch) | Account extra / app data / ad-unit extra → processed config keys |
| Init config | `internal/sdkapi/adapter_init_config.go` (+ store special cases) | Per-network init structs |
| Admin validators | `internal/admin` (e.g. line-item / demand-source account type switches) | Account type + extra validation |
| OpenAPI extras | `internal/admin/openapi` line-item extra schemas | Per-network extra field shapes |
| UI network defs | `web/bidon_ui/constants/Networks.js` (`NETWORK_DEFS`, bidding/waterfall key lists) | key, label, accountType, auction membership |
| UI demand-source options | `web/bidon_ui/constants/DemandSourceOptions.js` | Duplicate label / `DemandSourceAccount::*` list |
| Onboarding docs | `docs/network.md` | Tribal checklist of files to touch |
| Seeds | DB seed data | Sample apps / auctions / line items |

### Config remap patterns (today)

From `AdaptersConfigBuilder.Build` — typical shapes to capture in registry metadata:

| Pattern | Examples |
|---------|----------|
| Account extra → processed key (rename) | `publisher_id` → `seller_id` (bigoads, mintegral); `account_id` → `seller_id` (vungle) |
| App data key variants | `app_id` vs `app_key` (inmobi, moloco, zmaticoo) |
| Ad-unit extra → `tag_id` / `placement_id` | `slot_id`, `unit_id`, `ad_unit_id`, `placement_id`, `tag_id` |
| Env / demand secrets | Meta `app_secret` / `platform_id`; Moloco `api_key` |
| Special / non-remap | Amazon `price_points_map`; Adikteev `sdk_instance_id`; BidMachine mediation fields |
| Default fallback | `default:` copies raw `extra` |

### UI duplication

- `Networks.js` already claims to be a “canonical” UI registry (`NETWORK_DEFS` + `WATERFALL_NETWORK_KEYS` / `BIDDING_NETWORK_KEYS`).
- `DemandSourceOptions.js` re-lists labels and STI account types by hand.
- Backend runtime does not read either file.

---

## Design (proposed)

### Go registry (source of truth)

Prefer something like `internal/adapter/registry` (name TBD) with entries for each network:

```go
type Network struct {
	Key              Key
	Label            string
	AccountType      string // e.g. "DemandSourceAccount::Moloco"
	SupportsBidding  bool
	SupportsWaterfall bool

	// Declarative remaps for AdaptersConfigBuilder (v1)
	AccountExtraMaps map[string]string // source JSON key → processed key
	AppDataMaps      map[string]string
	AdUnitExtraMaps  map[string]string // e.g. "unit_id" → "tag_id"

	// Optional capability / escape hatches
	UsesEnvSecrets   bool // Meta, Moloco — keep injection explicit in builder
	// ProprietaryBidder, CustomInit, etc. as needed
}
```

Exact field set is finalized in BAC-46; the important contract is: **one table drives membership + standard remaps**.

### Runtime consumers

- `AdaptersConfigBuilder` applies registry remaps; only true specials (env secrets, Amazon price points, BidMachine mediation) stay as explicit code.
- `biddingAdapters` map can remain a builder registry, but membership / metadata should align with `SupportsBidding`.
- amazon: described in catalog; keep proprietary `FetchBids` path.

### Admin + UI

1. **Interim (BAC-48):** derive `DemandSourceOptions` from `NETWORK_DEFS` (one local UI list).
2. **Catalog API (BAC-49):** read-only admin endpoint wrapping the Go registry.
3. **UI (BAC-50):** fetch catalog; delete static duplicated maps.

---

## Implementation steps (Linear children)

1. [BAC-46](https://linear.app/bidon/issue/BAC-46/inventory-network-registration-surfaces-and-design-registry-shape) — Inventory freeze + finalize registry schema (this doc is the starting point).
2. [BAC-47](https://linear.app/bidon/issue/BAC-47/introduce-go-network-registry-and-migrate-runtime-config-wiring) — Introduce Go registry; migrate `AdaptersConfigBuilder` (and related runtime wiring).
3. [BAC-48](https://linear.app/bidon/issue/BAC-48/collapse-ui-demandsourceoptions-onto-networksjs-registry) — Collapse UI `DemandSourceOptions` onto `Networks.js`.
4. [BAC-49](https://linear.app/bidon/issue/BAC-49/expose-admin-api-network-catalog-from-go-registry) — Admin API network catalog endpoint.
5. [BAC-50](https://linear.app/bidon/issue/BAC-50/drive-admin-ui-networks-from-catalog-api-delete-static-maps) — UI consumes catalog; delete static maps.
6. [BAC-51](https://linear.app/bidon/issue/BAC-51/update-network-onboarding-docs-to-registry-checklist) — Rewrite `docs/network.md` checklist.

---

## Acceptance criteria

- [ ] Adding a network primarily means registering it once (plus adapter implementation / seeds).
- [ ] Standard account/app/ad-unit field remaps are data-driven from the Go registry (no unbounded switch for ordinary cases).
- [ ] UI network / account-type dropdowns are not maintained as duplicate static lists.
- [ ] Admin catalog API exposes registry data for the UI.
- [ ] `docs/network.md` reflects the post-registry onboarding path.
- [ ] Branch: `feature/BAC-45/network-registry` from `new-main`.

---

## Follow-ups after merge

1. Declarative (or generated) per-network admin form / OpenAPI extra schemas.
2. Align admin line-item validators with registry metadata where safe.
3. BAC-21 typed demand context on the SDK auction path.
4. Normalize inventory ID names at rest (reduce remap tables over time).
