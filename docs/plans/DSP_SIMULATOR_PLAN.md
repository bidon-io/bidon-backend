# DSP OpenRTB Response Simulator — Implementation Plan

## Overview

A standalone HTTP endpoint that receives OpenRTB 2.x bid requests, inspects the `imp[]` array to determine the requested impression type and dimensions, and returns a matching pre-baked bid response from files. The simulator acts as a mock DSP, returning realistic openRTB bid responses without making real network calls.

## Data Source

Existing `_bidres.json` files under `internal/sdkapi/v2/apihandlers/testdata/auction/adikteev/` — 12 real OpenRTB 2.x bid response fixtures, all for the `adikteev` DSP.

### Available Files

| File | Key |
|------|-----|
| `android_adikteev_banner_bidres.json` | `adikteev/android/banner/320x50` |
| `ios_adikteev_banner_bidres.json` | `adikteev/ios/banner/320x50` |
| `android_adikteev_banner_mrec_bidres.json` | `adikteev/android/banner/300x250` |
| `ios_adikteev_banner_mrec_bidres.json` | `adikteev/ios/banner/300x250` |
| `android_adikteev_interstitial_bidres.json` | `adikteev/android/banner/320x480` |
| `ios_adikteev_interstitial_bidres.json` | `adikteev/ios/banner/320x480` |
| `android_adikteev_interstitial_mraid_bidres.json` | `adikteev/android/banner/320x480/mraid` |
| `ios_adikteev_interstitial_mraid_bidres.json` | `adikteev/ios/banner/320x480/mraid` |
| `android_adikteev_rewarded_vast_bidres.json` | `adikteev/android/video/vast` |
| `ios_adikteev_rewarded_vast_bidres.json` | `adikteev/ios/video/vast` |
| `android_adikteev_rewarded_native_bidres.json` | `adikteev/android/native/0x0` |
| `ios_adikteev_rewarded_native_bidres.json` | `adikteev/ios/native/0x0` |

> **Note:** `vast` keys carry no dimensions. The `rewarded_vast` fixtures' bid creative (`360x640`) doesn't match the dimensions of the video imp that would realistically request it (`320x480`) — see [Dimension Extraction](#dimension-extraction-exact-match-only).

## Matching Logic

The lookup key combines fields extracted from the incoming OpenRTB bid request:

```
{displaymanager}_{device.os}_{imp_media_type}_{w}x{h}[_{variant}]
```

### Sources for Each Component

| Component | Source | Example Value |
|---|---|---|
| `displaymanager` | `imp.displaymanager` | `"adikteev"` |
| `device.os` | `request.device.os` | `"android"`, `"ios"` |
| `media_type` | Which OpenRTB media object is present in the imp | `"banner"`, `"video"`, `"native"` |
| `w×h` | Exact dimensions from the present media object (omitted entirely for the `vast` variant — see below) | `320x50` (from `banner.w`/`banner.h`), `320x480` (from `video.w`/`video.h`) |
| `variant` | Optional creative type suffix detected from imp fields | `"mraid"`, `"vast"` |

### Media Type Detection (from imp object)

- `imp.banner != nil` → `"banner"` type
- `imp.video != nil` → `"video"` type
- `imp.native != nil` → `"native"` type

### Variant Detection (from imp fields)

| Condition | Variant |
|---|---|
| `banner.api` contains MRAID framework IDs (3 or 5) | `"mraid"` |
| `video.mimes` contains VAST-compatible MIME types | `"vast"` |

`native` is not a variant — it is its own media type (see Media Type Detection above), matched independently of `banner`/`video`.

### Dimension Extraction (exact match only)

- **Banner**: Read `banner.W` and `banner.H` (both `*int64`)
- **Video**: Read `video.W` and `video.H` (both plain `int64`) — **except** when the `vast` variant is detected, in which case dimensions are dropped from the key entirely (key is `{displaymanager}_{device.os}_video_vast`, no `{w}x{h}` segment). Real DSPs commonly return a VAST creative sized differently than the requested player (e.g. the bundled `rewarded_vast` fixtures request `320x480` but the creative itself is `360x640`), so exact dimension matching is not meaningful for this variant.
- **Native**: No dimensions in the imp request object — uses `0x0` as placeholder

No fuzzy or range matching on dimensions otherwise. Banner and non-vast video/native matches remain exact.

### Multi-Format Imps

When a single imp has multiple media objects (e.g., both `video` and `native`), each media type is matched independently and all matching responses are included in the result.

### Multi-Imp Requests

Each imp in the request array is processed independently. Results from all matched imps are aggregated into a single `BidResponse`. If no imp matches any response, a `204 No Content` (no-bid) is returned.

### Multiple Fixtures Per Key

The response store indexes fixtures as `map[string][]*openrtb2.BidResponse` — a key may resolve to more than one candidate file (this also covers cases where multiple fixtures are deliberately added under the same key for response variety). When a key has multiple candidates, one is chosen uniformly at random per request. Tests that need determinism should assert membership in the candidate set rather than a single expected fixture.

### Response ID Rewriting

Fixture files hardcode `BidResponse.id` and each `bid.impid` from the original request that was captured to produce them. Before returning a matched fixture, the service rewrites:
- `BidResponse.id` → the incoming request's `id`
- each returned `bid.impid` → the `id` of the imp it was matched against

This keeps responses OpenRTB-valid for whatever live request they answer, regardless of which fixture request originally produced them.

## Architecture

### Package Structure

```
cmd/bidon-dspsimulator/main.go          # Binary entrypoint
internal/dspsimulator/
├── service.go                           # Core Service struct: orchestrates matching
├── matcher.go                          # Imp inspection, key construction, variant detection
├── response_store.go                   # File scanning, JSON loading, key → response indexing
├── server.go                          # HTTP handler: POST /bid
└── service_test.go                    # Integration tests with real fixtures
```

### Component Responsibilities

**`cmd/bidon-dspsimulator/main.go`**
- Reads `DSP_RESPONSE_DIR` and `DSP_SIMULATOR_PORT` from environment
- Initializes the `ResponseStore` (loads all `_bidres.json` files)
- Starts the HTTP server on the configured port

**`internal/dspsimulator/response_store.go`**
- Scans configured directory for `*_bidres.json` files
- Parses each file into `openrtb2.BidResponse`
- Derives the lookup key from the filename convention (dims omitted for the `vast` variant — see [Dimension Extraction](#dimension-extraction-exact-match-only))
- Stores responses in an in-memory map keyed by lookup key, with a slice of candidates per key (`map[string][]*openrtb2.BidResponse`) to support [multiple fixtures per key](#multiple-fixtures-per-key)
- Picks one candidate at random when a key has more than one
- Returns `nil` when a key is not found (no-bid signal)

**`internal/dspsimulator/matcher.go`**
- Receives an `openrtb2.BidRequest`
- Iterates each `imp` in the request
- For each imp: extracts `displaymanager`, reads `device.os` from the request, detects media type, extracts exact `w`×`h` (omitted for `vast`), detects variant
- Constructs the lookup key
- Returns a list of keys to query the store

**`internal/dspsimulator/service.go`**
- `Service` struct wraps `ResponseStore` and the matcher
- `HandleBidRequest()` method: receives `*openrtb2.BidRequest`, builds keys, queries store, aggregates results into `*openrtb2.BidResponse`
- Rewrites `BidResponse.id` and each matched `bid.impid` to the live request/imp before returning (see [Response ID Rewriting](#response-id-rewriting))
- Returns `nil` if no matches found (→ 204 No Content)

**`internal/dspsimulator/server.go`**
- HTTP handler for `POST /bid`
- Accepts `openrtb2.BidRequest` JSON body
- Calls `Service.HandleBidRequest()`
- Returns `openrtb2.BidResponse` JSON (200 OK) or `204 No Content`
- Handles parse errors with 400 Bad Request

## HTTP Endpoint

```
POST /bid
Content-Type: application/json

Request Body:  openrtb2.BidRequest (JSON)
Response Body: openrtb2.BidResponse (JSON)  —or—  204 No Content
```

## Configuration

Environment variables:

| Variable | Default                                                      | Description |
|---|--------------------------------------------------------------|---|
| `DSP_RESPONSE_DIR` | `./internal/sdkapi/v2/apihandlers/testdata/auction/adikteev` | Directory containing `_bidres.json` files |
| `DSP_SIMULATOR_PORT` | `1326`                                                       | HTTP listen port |

## Response File Key Derivation

Keys are derived at load time from the `_bidres` filenames (not from inspecting file contents). The filename convention is:

```
{os}_{dsp}_{descriptor}_bidres.json
```

`os` = `android` or `ios`; `dsp` = the adapter key (e.g. `adikteev`); `descriptor` maps directly to a `(media_type, variant)` pair — it is not decomposed generically, since the ad-unit name in the filename (`interstitial`, `rewarded`) doesn't always correspond 1:1 with the OpenRTB media type it uses:

| Descriptor | media_type | variant | Dimensions |
|---|---|---|---|
| `banner` | `banner` | — | from `bid.w`/`bid.h` |
| `banner_mrec` | `banner` | — | from `bid.w`/`bid.h` |
| `interstitial` | `banner` | — | from `bid.w`/`bid.h` |
| `interstitial_mraid` | `banner` | `mraid` | from `bid.w`/`bid.h` |
| `rewarded_vast` | `video` | `vast` | omitted (see [Dimension Extraction](#dimension-extraction-exact-match-only)) |
| `rewarded_native` | `native` | — | `0x0` placeholder |

New descriptors added for future fixtures must be added to this table explicitly.

## Extending to Other DSPs

To add support for additional DSPs, place their `_bidres.json` files in the configured response directory. The service automatically picks up any files matching the `*_bidres.json` pattern. Each file is parsed into an `openrtb2.BidResponse` and indexed by its composite key.

New files should follow the same naming convention: `{os}_{dsp_key}_{descriptor}_bidres.json`, and any new descriptor must be added to the [Response File Key Derivation](#response-file-key-derivation) table so its `(media_type, variant)` mapping is explicit. Multiple files may share the same descriptor/key to provide response variety — see [Multiple Fixtures Per Key](#multiple-fixtures-per-key).

## Test Strategy

- **Unit tests** for `matcher.go`: verify key construction from various `Imp` shapes (banner only, video only, native only, multi-format, vast key omits dims)
- **Unit tests** for `response_store.go`: verify file loading, parsing, key mapping, and that a key with multiple candidate files returns one of them (assert membership in the candidate set, not a single expected fixture — selection is random)
- **Integration tests**: send real `_bidreq` JSON files, verify matching `_bidres` JSON is returned (for keys with multiple candidates, assert the response matches one of the expected set)
- **ID rewriting tests**: verify `BidResponse.id` and each `bid.impid` in the returned response match the live request's `id`/imp `id`, not the fixture's original captured values
- **Edge cases**: empty imp array → 204, unknown displaymanager → 204, missing dimensions → 204, multi-imp aggregation

## Files to Create

| File | Purpose |
|------|---------|
| `cmd/bidon-dspsimulator/main.go` | Binary entrypoint (~30 lines) |
| `internal/dspsimulator/service.go` | Core matching service (~50 lines) |
| `internal/dspsimulator/matcher.go` | Imp inspection + key construction (~80 lines) |
| `internal/dspsimulator/response_store.go` | File scanning + response loading (~60 lines) |
| `internal/dspsimulator/server.go` | HTTP handler (~60 lines) |
| `internal/dspsimulator/service_test.go` | Integration tests (~120 lines) |
