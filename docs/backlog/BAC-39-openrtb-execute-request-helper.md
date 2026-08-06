# Backlog: Shared OpenRTB ExecuteRequest HTTP helper

**Status:** In Progress (inventory frozen)  
**Linear:** [BAC-39](https://linear.app/bidon/issue/BAC-39/shared-openrtb-executerequest-http-helper)  
**Branch:** `feature/BAC-39/openrtb-execute-request-helper`  
**Base:** Current OpenRTB stack tip (after BAC-26 / BAC-27 / BAC-28 work)  
**Related:** BAC-26 (parse), BAC-27 (`BuildRTBRequest`), BAC-28 (creative builders)

---

## Summary

OpenRTB demand adapters each rebuild nearly identical `ExecuteRequest` HTTP transport: marshal the bid request, POST JSON, read the body, and fill `DemandResponse` Status / RawResponse / RawRequest. Only URL selection, auth / proprietary headers, query mutation, and a few post-response side channels differ by network.

This work extracts a **shared ExecuteRequest HTTP helper** so adapters supply endpoint, headers, and network quirks. Bid parsing stays in BAC-26; CreateRequest shell / creatives stay in BAC-27 / BAC-28.

---

## Motivation

- Remove ~35–50 lines of duplicated marshal → POST → read-body scaffolding across ~14 OpenRTB adapters.
- Make new OpenRTB networks “URL + headers + helper” instead of copy-paste.
- Keep truly non-standard demands on an explicit custom path (amazon `FetchBids`).
- Completes the OpenRTB lifecycle cleanup next to shared parse (BAC-26) and CreateRequest shell (BAC-27).

---

## Current inventory

### Shared scaffolding (repeated almost everywhere)

| Step | Typical behavior |
|------|------------------|
| Seed `DemandResponse` | `DemandID`, `RequestID`, optional TagID / PlacementID / TimeoutURL / ImpID |
| Marshal request | `json.Marshal` → `RawRequest` (or Error + return) |
| Build HTTP request | `POST`, body = marshaled JSON |
| Headers | Always `Content-Type: application/json`; some add OpenRTB version / auth |
| Execute | `client.Do` → Error on failure |
| Read body | `io.ReadAll` → `RawResponse` + `Status` |

### Migration tiers

| Tier | Adapters | Approach |
|------|----------|----------|
| A — Easy (fixed URL) | vkads, yandex, mobilefuse, bigoads, adikteev, vungle, inmobi, mintegral | Drop-in `ExecuteRTBRequest` with static URL + optional Headers |
| B — Medium (geo URL) | bidmachine, taurusx, zmaticoo, moloco | Resolve URL/auth locally; pass URL + Headers (+ ImpID for zmaticoo) |
| C — Hard (hooks) | startio, meta | Prebuilt URL or `PrepareURL` (startio query); Meta `TimeoutURL` + `AfterDo` for `X-Fb-An-Errors` |
| D — Out of scope | amazon | Keep `FetchBids`; no `ExecuteRequest` |

### Network-specific overrides (stay in adapters)

| Pattern | Adapters |
|---------|----------|
| Fixed endpoint URL | Easy tier |
| Geo / region endpoint maps | bidmachine, taurusx, zmaticoo, moloco, startio |
| `X-OpenRTB-Version: 2.5` | vungle, inmobi, taurusx, zmaticoo |
| `openrtb: 2.5` header | mintegral |
| `Authorization` API key | moloco |
| Query mutation (`account`, `testAdsEnabled`) | startio |
| Path embeds PlatformID + `TimeoutURL` + 400 error header | meta |
| Stash `ImpID` on DemandResponse | zmaticoo |
| Proprietary non-OpenRTB path | amazon |

---

## Design (proposed)

### Shared helper / options

Shared API: `adapters.ExecuteRTBRequest` with `adapters.ExecuteRTBOptions` (e.g. in `rtb_execute.go` next to `rtb_request.go`).

```go
func CountryFromRequest(request openrtb.BidRequest) string

type ExecuteRTBOptions struct {
	DemandID    adapter.Key
	URL         string
	TagID       string
	PlacementID string
	TimeoutURL  string
	ImpID       string
	Headers     http.Header
	PrepareURL  func(base string, request openrtb.BidRequest) (string, error)
	AfterDo     func(resp *http.Response, dr *DemandResponse)
}

func ExecuteRTBRequest(
	ctx context.Context,
	client *http.Client,
	request openrtb.BidRequest,
	opts ExecuteRTBOptions,
) *DemandResponse
```

**Behavior:**

1. Seed `DemandResponse` from opts + `request.ID`
2. Marshal → `RawRequest` (fail → return with `Error`)
3. If `PrepareURL != nil`, replace URL; else use `opts.URL`
4. Empty URL → `Error`
5. Build POST, set `Content-Type: application/json`, merge `Headers`
6. `Do` + read body → `Status` / `RawResponse`
7. Call `AfterDo` if set

### Do not force one path

Prefer `ExecuteRTBRequest` + local URL/header prep over a rigid DSL. amazon stays on `FetchBids`.

### Boundary with other BAC work

```
CreateRequest (BAC-27/28) → ExecuteRequest (BAC-39) → ParseBids (BAC-26)
```

---

## Out of scope

- Shared creative builders / CreateRequest shell (BAC-27 / BAC-28)
- Parse / enrich path (BAC-26)
- Reworking amazon `FetchBids`
- Unifying per-network geo endpoint maps into one router (optional later)

---

## Implementation steps (Linear children)

1. [BAC-40](https://linear.app/bidon/issue/BAC-40/inventory-executerequest-scaffolding-across-openrtb-adapters) — Inventory freeze; confirm Easy / Medium / Hard / Out-of-scope tiers.
2. [BAC-41](https://linear.app/bidon/issue/BAC-41/introduce-shared-executertbrequest-api) — Introduce `ExecuteRTBRequest` + `CountryFromRequest` with unit tests.
3. [BAC-42](https://linear.app/bidon/issue/BAC-42/migrate-easy-fixed-url-adapters-onto-executertbrequest) — Migrate easy fixed-URL adapters.
4. [BAC-43](https://linear.app/bidon/issue/BAC-43/migrate-geo-routed-adapters-onto-executertbrequest) — Migrate geo-routed adapters.
5. [BAC-44](https://linear.app/bidon/issue/BAC-44/migrate-startio-meta-onto-executertbrequest-leave-amazon-alone) — Migrate Start.io + Meta; leave amazon alone.

---

## Acceptance criteria

- [ ] Common ExecuteRequest HTTP transport lives in `adapters.ExecuteRTBRequest`.
- [ ] Migrated adapters only set network-specific differences (URL, headers, hooks, DR metadata).
- [ ] Existing ExecuteRequest tests still pass (or are updated against the shared helper).
- [ ] amazon / non-standard paths are not forced onto `ExecuteRTBRequest`.
- [ ] Branch: `feature/BAC-39/openrtb-execute-request-helper` from the current OpenRTB stack tip.
- [ ] Complements BAC-26 (parse) and BAC-27 (`BuildRTBRequest`); does not re-implement CreateRequest or bid parsing.

---

## Follow-ups after merge

1. Optionally unify geo endpoint selection where product-safe.
2. Optional CreateRequest micro-helpers (`ImpForAdType`, `DemandToken`) as a separate ticket.
3. New OpenRTB networks: `BuildRTBRequest` + `ExecuteRTBRequest` + `ParseDemandResponse` as the default path.
