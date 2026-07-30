# Backlog: Shared OpenRTB banner / interstitial / rewarded creative builders

**Status:** In Progress  
**Linear:** [BAC-28](https://linear.app/bidon/issue/BAC-28/shared-openrtb-banner-interstitial-rewarded-creative-builders)  
**Branch:** `feature/BAC-28/openrtb-shared-creative-builders`  
**Base:** `feature/BAC-27/openrtb-create-request-imp-shell` (after `BuildRTBRequest` shell)  
**Blocked by:** BAC-27 (done)  
**Related:** BAC-26 (parse), BAC-27 (`BuildRTBRequest`), BAC-19 (rendering)

---

## Summary

OpenRTB adapters repeatedly copy local `bannerFormats` maps, adaptive→leaderboard sizing, `FullscreenFormats` usage, MRAID/API flags, and similar banner / interstitial / rewarded Imp wiring. Creative shapes still diverge by network (dual banner+video, fixed 1920×1080 rewarded, demand-specific APIs), so this must not become one forced mega-builder.

This work centralizes **shared size maps and optional creative helpers** for the boring cases. Adapters keep ownership of demand-specific creative quirks. Complements BAC-27: `BuildRTBRequest` owns request scaffolding; BAC-28 owns creative object construction fed into that shell.

---

## Motivation

- Remove ~14 copy-pasted `bannerFormats` tables and near-identical banner helpers.
- Push more adapters onto already-shared `FullscreenFormats` (and shared orientation swap) instead of re-implementing sizing.
- Give optional helpers for typical banner / interstitial / rewarded Imps without erasing network-specific shapes.
- Keep divergent networks (adikteev dual creative, bigoads minimal interstitial/rewarded, vkads/yandex custom rewarded sizes, etc.) on explicit custom builders.

---

## Current inventory

### Already shared

| Helper | Location | Used by |
|--------|----------|---------|
| `FullscreenFormats` (`PHONE` 320×480, `TABLET` 768×1024) | `adapters/helpers.go` | Most interstitial/rewarded paths (moloco, meta, mintegral, vungle, startio, inmobi, taurusx, zmaticoo, bidmachine, adikteev, …) |
| `BannerFormats` + `ResolveBannerSize` | `adapters/helpers.go` | Tier A banner paths (moloco, meta†, mintegral, vungle, startio, inmobi, bidmachine, mobilefuse, yandex, vkads, taurusx, zmaticoo). †meta keeps adaptive phone width `0` override. |

### Repeated per adapter

| Pattern | Typical adapters | Notes |
|---------|------------------|--------|
| Local `bannerFormats` map (320×50, 728×90, 300×250, adaptive/empty defaults) | Nearly all OpenRTB adapters | Slight key-set differences (adikteev: banner+MREC only; bigoads: no leaderboard) |
| Adaptive tablet → leaderboard size | moloco, meta, mintegral, vungle, startio, inmobi, bidmachine, mobilefuse, yandex, vkads, … | bigoads **errors** on adaptive tablet instead |
| Banner Imp: `Instl:0`, `Pos: AboveFold`, W/H from map | Most | API frameworks sometimes added (inmobi, adikteev) |
| Interstitial: `Instl:1`, fullscreen size + orientation swap, banner (+ often video) | Most using `FullscreenFormats` | bigoads/mobilefuse/yandex/vkads use thinner or custom shapes |
| Rewarded: fullscreen + video / skip / ext flags | Most | vkads uses fixed 1920×1080; adikteev reuses interstitial; bigoads video-only minimal |
| Dual banner+video interstitial/rewarded | adikteev, moloco, startio, bidmachine, … | MIME / protocol lists differ |
| MRAID / API framework lists | adikteev (`MRAIDAPI`), inmobi (numeric APIs), others omit | Should stay optional, not forced |

### Migration tiers (proposed)

| Tier | Focus | Adapters / approach |
|------|-------|---------------------|
| A — shared size maps | Introduce `BannerFormats` (or similar) + adaptive→leaderboard helper; delete local maps where identical | Straightforward banner maps: moloco, meta, mintegral, vungle, startio, inmobi, bidmachine, mobilefuse, yandex, vkads, taurusx, zmaticoo |
| B — optional creative helpers | Thin helpers for typical banner / interstitial / rewarded Imps (size + Instl + Pos + optional video defaults) | Adopt where creative shape matches; keep MIME/API as options or post-mutate |
| C — keep custom | Divergent creative wiring | adikteev (dual + MRAID), bigoads (no leaderboard / minimal FS), vkads rewarded size, others as needed |
| D — out of scope | Non-OpenRTB / no CreateRequest creative path | amazon |

---

## Design (proposed)

### Shared size maps first

1. Promote a package-level banner size map next to `FullscreenFormats` (name TBD, e.g. `BannerFormats`).
2. Shared helper for “resolve banner WxH” including adaptive tablet → leaderboard (with opt-out / error path for bigoads).
3. Keep `FullscreenFormats`; optionally add orientation-aware size helper used by interstitial/rewarded.

### Optional creative builders (after maps)

Something like `BuildBannerImp` / `BuildInterstitialImp` / `BuildRewardedImp` (names TBD) that:

- Produce an `*openrtb2.Imp` creative body only (no TagID / bidfloor / display manager — those stay in `BuildRTBRequest`).
- Accept options for API frameworks, video MIME lists, Instl overrides, skip, ext blobs.
- Return Imp suitable to pass into `BuildRTBRequest`.

Adapters that do not fit call custom local builders and still use `BuildRTBRequest` for the shell.

### Boundary with BAC-27

```
creative Imp  →  BuildRTBRequest(shell)  →  network overrides (token/ext)
     ↑ BAC-28              ↑ BAC-27
```

---

## Out of scope

- Changing `BuildRTBRequest` contract beyond consuming shared creatives
- Forcing every adapter onto one creative builder
- amazon `FetchBids`
- Parse / rendering (BAC-26 / BAC-19)

---

## Implementation steps (Linear children)

1. [BAC-34](https://linear.app/bidon/issue/BAC-34/inventory-openrtb-creative-size-maps-and-impression-shapes) — Inventory freeze; confirm size maps / creative shape tiers. **Done**
2. [BAC-35](https://linear.app/bidon/issue/BAC-35/introduce-shared-bannerformats-map-and-size-resolve-helper) — Shared `BannerFormats` map + resolve helper with unit tests. **Done**
3. [BAC-36](https://linear.app/bidon/issue/BAC-36/migrate-adapters-onto-shared-banner-size-map) — Migrate adapters onto shared banner sizes; delete duplicate maps. **Done** (adikteev / bigoads kept custom)
4. [BAC-37](https://linear.app/bidon/issue/BAC-37/introduce-optional-bannerinterstitialrewarded-creative-helpers) — Optional banner / interstitial / rewarded creative helpers.
5. [BAC-38](https://linear.app/bidon/issue/BAC-38/migrate-common-adapters-onto-creative-helpers-keep-divergent-custom) — Migrate common adapters; keep divergent custom.
6. **Sweep** — amazon untouched; CreateRequest tests green.

---

## Acceptance criteria

- [x] Repeated banner/fullscreen size tables are shared instead of copy-pasted per adapter.
- [ ] Common creative defaults have one home; network-specific creatives remain in adapters.
- [x] No forced single builder for every demand.
- [x] Existing CreateRequest / creative adapter tests still pass.
- [x] Branch: `feature/BAC-28/openrtb-shared-creative-builders` from `feature/BAC-27/openrtb-create-request-imp-shell`.
- [x] Complements `BuildRTBRequest` (BAC-27); creative builders do not re-implement shell fields.

---

## Follow-ups after merge

1. Optionally unify video MIME / protocol defaults where product-safe.
2. Optionally fold remaining near-duplicates (orientation swap, Pos defaults) into helpers.
3. New OpenRTB networks: creative helper + `BuildRTBRequest` as the default path.
