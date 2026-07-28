# Backlog: Centralize OpenRTB bid response parsing

**Status:** Backlog  
**Linear:** [BAC-26](https://linear.app/bidon/issue/BAC-26/centralize-openrtb-parsebids-at-the-builder-with-optional-network)  
**Branch:** `feature/BAC-26/openrtb-parse-bids-dto`  
**Base:** `new-main`  
**Blocks:** BAC-19 (uSDK rendering config) — rebase BAC-19 on top after this lands  

---

## Summary

OpenRTB demand adapters each implement nearly identical `ParseBids` logic: HTTP status handling, unmarshalling the raw response, taking the first seat/bid, and mapping standard fields into `NormalizedBid`. Only a few networks add custom extraction (e.g. signaldata, alternate payload source). Proprietary adapters (non-OpenRTB) need a full custom parse path.

This work moves the common OpenRTB parse to the **bidding builder call site**, so most networks no longer define `ParseBids` at all. Networks only opt in when they need enrichment or a fully custom parser.

---

## Motivation

- Remove duplicated parse/mapping across ~13 OpenRTB adapters.
- Make the default path “request + execute only” for new OpenRTB networks.
- Keep network-specific extraction as a small optional override.
- Land on `new-main` before BAC-19 so rendering enrichment continues to run once after a single shared parse, and BAC-19 can rebase onto a thinner adapter surface.

---

## Current inventory (customization only)

| Pattern | Adapters |
|--------|----------|
| Standard OpenRTB map only | adikteev, bidmachine, bigoads, meta, vkads, vungle, inmobi, mintegral, moloco, startio |
| Extra: signaldata from bid ext | yandex, mobilefuse |
| Extra: payload from bid response ext (not adm) | taurusx |
| Empty seatbid → no bid | inmobi, mintegral, yandex, taurusx |
| Empty seatbid → error | moloco, startio |
| Empty seatbid unchecked (panic risk) | several “standard” adapters |
| Non-OpenRTB custom parse | zmaticoo |
| Separate fetch path (unchanged) | amazon (`FetchBids`) |

Also: **taurusx** and **zmaticoo** currently parse inside `ExecuteRequest`. That must stop so the builder owns a single parse step.

---

## Design

### Call site owns parsing

In `bidding.Builder.processAdapter`, after `ExecuteRequest` succeeds, call a shared framework entrypoint (name TBD, e.g. `ParseDemandResponse`) instead of requiring `Adapter.ParseBids` on every network.

Pipeline order:

1. `ExecuteRequest` — status + raw body only  
2. Shared parse / optional custom parse / optional enrich  
3. Existing post-parse enrichment (BAC-19: `EnrichBid` / rendering) after rebase  

### Required bidder interface shrinks

`BidderInterface` keeps `CreateRequest` and `ExecuteRequest` only.  
`ParseBids` is **removed** from the required interface.

### Optional capabilities (type-asserted at the call site)

1. **Custom bid parser** — proprietary / non-OpenRTB networks (zmaticoo). Full ownership of parsing the raw response into `DemandResponse`.
2. **OpenRTB bid enricher** — networks that need extra fields after the default OpenRTB map (yandex, mobilefuse, taurusx). Receives the demand response plus the parsed OpenRTB bid response / seat / bid and mutates the DTO.

Most OpenRTB adapters implement **neither**.

### Shared OpenRTB path responsibilities

- Standard HTTP status handling (align on the common 204 / auth-style failures / 200 set).
- Unmarshal raw response to OpenRTB bid response.
- Empty seat/bid policy — **default: treat as no-bid** (also fixes unchecked indexing). Decide whether moloco/startio keep an error policy via enricher/custom path or adopt no-bid.
- Map standard fields into `NormalizedBid` (ids, price, adm payload, seat, notice URLs, raw bid ext for later enrichment).
- Use `DemandResponse.DemandID` already set by execute; do not hardcode demand keys in the mapper.
- Invoke optional OpenRTB enricher when present.

### Network outcomes

| Network | After change |
|--------|----------------|
| Most OpenRTB adapters | No parse-related methods |
| yandex, mobilefuse | OpenRTB enricher for signaldata |
| taurusx | OpenRTB enricher for payload-from-response-ext; stop parsing in `ExecuteRequest` |
| zmaticoo | Custom bid parser; stop parsing in `ExecuteRequest` |
| amazon | Unchanged (`FetchBids` bypass) |

---

## Out of scope

- Rendering / `EnrichBid` (BAC-19) — remains a separate post-parse step at the builder.
- Auction-aware rendering defaults (e.g. container format from ad type).
- Reworking amazon’s fetch path.
- Changing CreateRequest / ExecuteRequest contracts beyond “execute must not parse”.

---

## Implementation steps

1. Add shared parse entrypoint + optional enricher / custom-parser contracts; unit-test status, bad JSON, empty seatbid policy, default field map, enricher invocation.
2. Switch `Builder.processAdapter` to the shared entrypoint.
3. Remove `ParseBids` from the required bidder interface.
4. Delete parse methods from standard OpenRTB adapters; add enrichers only where needed.
5. Stop parse-inside-`ExecuteRequest` for taurusx and zmaticoo; keep zmaticoo on the custom parser path.
6. Move per-adapter ParseBids tests to framework tests + enricher-specific tests.
7. Merge to `new-main`, then rebase `feature/BAC-19/usdk-rendering-config`.

---

## Acceptance criteria

- [x] Required bidder interface no longer includes `ParseBids`.
- [x] Most OpenRTB networks define zero parse-related methods.
- [x] Network-specific extraction exists only as optional OpenRTB enrichers (or custom parsers for non-OpenRTB).
- [x] Single parse path in the builder; `ExecuteRequest` does not parse bids.
- [x] Empty seatbid no longer panics on migrated OpenRTB adapters.
- [x] Existing parse behavior covered by framework / enricher tests.
- [x] Branch cut from `new-main`: `feature/BAC-26/openrtb-parse-bids-dto`.
- [x] Linear issue created; BAC-19 marked blocked by this issue.

---

## Follow-ups after merge

1. Rebase BAC-19; keep rendering enrichment immediately after the shared parse.
2. Optionally unify empty-seatbid error vs no-bid policy for moloco/startio.
3. Later: auction-context defaults on the same post-parse enrichment path.
