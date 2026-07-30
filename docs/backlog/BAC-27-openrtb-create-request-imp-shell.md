# Backlog: Shared OpenRTB CreateRequest impression shell

**Status:** Backlog  
**Linear:** [BAC-27](https://linear.app/bidon/issue/BAC-27/shared-openrtb-createrequest-impression-shell)  
**Branch:** `feature/BAC-27/openrtb-create-request-imp-shell`  
**Base:** `feature/BAC-26/openrtb-parse-bids-dto` (not `new-main` — sits on the BAC-26 → BAC-19 OpenRTB stack)  
**Related:** BAC-26 (parse centralization), BAC-28 (creative builders — follow-up), BAC-19 (rendering; currently stacked on BAC-26 tip)

---

## Summary

OpenRTB demand adapters each rebuild nearly identical `CreateRequest` scaffolding: UUID imp ID, `secure`, bidfloor / currency, display manager + SDK version, single-imp slice, and `Cur: USD`. Only a subset of fields actually differ by network (token placement, tag/app/publisher IDs, imp/request ext, device/regs quirks).

This work extracts a **shared request + impression shell** so adapters supply only network-specific overrides. Creative object construction (banner / interstitial / rewarded) stays out of scope — that is BAC-28.

---

## Motivation

- Remove duplicated CreateRequest scaffolding across ~14 OpenRTB adapters.
- Make new OpenRTB networks “shell + overrides” instead of copy-paste.
- Keep truly non-standard demands on an explicit custom path (do not force one shape).
- Pair with BAC-26 (shared parse) and precede BAC-28 (shared creative builders) in the OpenRTB lifecycle cleanup.

---

## Current inventory

### Shared scaffolding (repeated almost everywhere)

| Field / step | Typical value | Notes |
|--------------|---------------|--------|
| `imp.ID` | new UUID | All OpenRTB adapters |
| `imp.Secure` | `1` | Most; vkads omits |
| `imp.BidFloor` | `CalculatePriceFloor(...)` | All |
| `imp.BidFloorCur` | `"USD"` | Most; missing on adikteev, bigoads, bidmachine, mobilefuse |
| `imp.DisplayManager` | adapter key string | Most; vkads omits |
| `imp.DisplayManagerVer` | `Adapters[key].SDKVersion` | Most; vkads omits |
| `request.Imp` | `[]Imp{*imp}` | All |
| `request.Cur` | `["USD"]` | All |
| Ad-type switch | banner / interstitial / rewarded | Creative body left to BAC-28 |

### Network-specific overrides (stay in adapters)

| Pattern | Adapters |
|--------|----------|
| `imp.TagID` from config | bigoads, meta, mintegral, mobilefuse, moloco, startio, taurusx, vungle, vkads, yandex (AdUnitID), inmobi/zmaticoo (PlacementID) |
| Token → `user.BuyerUID` | bigoads, inmobi, meta, mintegral, moloco, startio |
| Token → `user.Data[].Segment[].Signal` | yandex, mobilefuse |
| Token → `user.Ext.buyeruid` | vkads |
| Token → `imp.Ext` (`bid_token` / nested) | bidmachine, vungle |
| Token → `request.Ext` | taurusx (placement token), zmaticoo (`sdk_token`), meta (auth), vkads (`pid`), bidmachine (`ExtraParams`) |
| Token → `app.Ext` | adikteev (`sdkinstanceid`) |
| `app.ID` / `app.Publisher.ID` | most TagID networks; meta uses publisher ID = AppID |
| `imp.Ext` network blob | bigoads (`adtype` + `networkid`), yandex, others merging into existing ext |
| Orientation / app.Ext extras | mintegral |
| No DisplayManager / Secure | vkads |
| Proprietary / non-OpenRTB request path | amazon (`FetchBids` — leave alone) |

### Migration tiers

| Tier | Adapters | Approach |
|------|----------|----------|
| A — shell + light overrides | adikteev, inmobi, moloco, startio, mobilefuse | First migrations; prove options API |
| B — shell + richer overrides | bigoads, mintegral, meta, vungle, yandex, taurusx, zmaticoo, bidmachine | TagID / token / ext / app fields via options or post-shell mutate |
| C — partial / careful | vkads | Missing DisplayManager/Secure today; decide whether shell defaults apply or opt-out |
| D — out of scope | amazon | Keep `FetchBids`; no CreateRequest shell |

---

## Design (proposed)

### Shared builder / options

Introduce something like `adapters.BuildOpenRTBRequest` (name TBD) that:

1. Takes the base `openrtb.BidRequest`, auction request, demand key, and an already-built creative `Imp` (or empty Imp for BAC-28 later).
2. Applies the common shell: UUID ID, secure, bidfloor (+ currency), display manager + version, `Imp` slice, `Cur`.
3. Accepts an options / override struct for the common optional knobs without forcing every network through one mega-struct:
   - `TagID`
   - `AppID` / `PublisherID`
   - `BuyerUID` (or a small token-placement helper later)
   - `BidFloorCur` default `"USD"` with opt-out
   - `Secure` default `1` with opt-out
   - Optional hooks to merge `Imp.Ext` / `Request.Ext` / `App.Ext` / `User`

Adapters keep ownership of:

- Building the creative object (until BAC-28)
- Network-specific token placement and ext shapes
- Validation of required config (empty TagID, missing token, etc.)

### Do not force one path

If a demand is highly custom (amazon today; possibly vkads if defaults would change behavior), leave an explicit full `CreateRequest` or opt-out flags. Prefer “shell + mutate” over a rigid DSL.

### Relationship to BAC-28

BAC-28 owns shared banner / interstitial / rewarded creative builders. This ticket only standardizes the **shell around** whatever Imp the adapter (or later shared creative builder) produces.

---

## Out of scope

- Shared creative builders (BAC-28)
- Parse / enrich path (BAC-26 / BAC-19)
- Reworking amazon `FetchBids`
- Unifying token placement into a single abstraction unless it falls out naturally from the options struct (can be a follow-up)

---

## Implementation steps (Linear children)

1. [BAC-29](https://linear.app/bidon/issue/BAC-29/inventory-createrequest-scaffolding-across-openrtb-adapters) — Inventory freeze; confirm migration tiers A/B/C/D and BidFloorCur / Secure / DisplayManager inconsistencies.
2. [BAC-30](https://linear.app/bidon/issue/BAC-30/introduce-shared-openrtb-createrequest-impression-shell-api) — Introduce shared shell API (options + builder) with isolated unit tests.
3. [BAC-31](https://linear.app/bidon/issue/BAC-31/migrate-tier-a-adapters-onto-createrequest-shell) — Migrate tier A: adikteev, inmobi, moloco, startio, mobilefuse.
4. [BAC-32](https://linear.app/bidon/issue/BAC-32/migrate-tier-b-adapters-onto-createrequest-shell) — Migrate tier B: bigoads, mintegral, meta, vungle, yandex, taurusx, zmaticoo, bidmachine.
5. [BAC-33](https://linear.app/bidon/issue/BAC-33/decide-vkads-shell-defaults-vs-opt-out) — Decide vkads shell defaults vs opt-out; migrate accordingly.
6. **Sweep** — Ensure amazon untouched; leave creative construction for BAC-28.

---

## Acceptance criteria

- [ ] Common CreateRequest scaffolding lives in one shared helper/builder.
- [ ] Migrated adapters only set network-specific differences (token, tag/app IDs, ext, validation).
- [ ] Existing CreateRequest tests still pass (or are updated to assert against the shared shell).
- [ ] amazon / non-standard paths are not forced onto the shell.
- [ ] Creative banner/interstitial/rewarded construction remains adapter-local (BAC-28).
- [ ] Branch: `feature/BAC-27/openrtb-create-request-imp-shell` from `feature/BAC-26/openrtb-parse-bids-dto`.

---

## Follow-ups after merge

1. BAC-28 — shared banner / interstitial / rewarded creative builders on top of this shell.
2. Optional: unify token placement helpers (`BuyerUID` vs `user.Data` signal vs `imp.Ext` bid_token).
3. Optional: normalize BidFloorCur / Secure defaults across networks that currently omit them (only if product-safe).
