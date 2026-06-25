# Ad Types and Ad Formats

## Overview

Bidon defines two orthogonal ad concepts:

- **Ad Type** — High-level UX behavior of the ad unit. Used in the Admin UI
  to configure auctions and line items, and passed by apps/SDKs in auction
  requests. Three types: Banner, Interstitial, Rewarded.

- **Ad Format** — Display size/orientation variant relevant only to Banner
  ad units.

## Ad Types (UX behavior)

Ad types describe the in-app presentation and user interaction model.

### Banner

Controlled by the app. Persistent, non-intrusive, mostly static images.
Can occupy a portion of the screen at a configurable size (see Ad Formats).
Always carries an associated Ad Format.

### Interstitial

Full screen, closable after ~5 seconds (or configurable duration).
Controlled by the app. Can be a static image, a video, or a playable ad.
No associated Ad Format — always fullscreen.

### Rewarded

Full screen, user-controlled. User must complete watching the full duration
to receive an in-app reward (e.g., extra lives, coins). Closing early forfeits
the reward. No associated Ad Format — always fullscreen.

## Ad Formats (IAB/AdCOM)

Ad Formats specify the display size and rendering constraints for Banner ad
units. They are defined by the IAB AdCOM specification:

- `com.iabtechlab.adcom.v1.placement.Placement.DisplayPlacement.DisplayFormat`
  (`proto/adcom/proto/com/iabtechlab/adcom/v1/placement/placement.proto:175`)

A `DisplayFormat` describes:

- Absolute width/height (`w`, `h`)
- Ratio-based sizing (`wratio`, `hratio`)
- Expandable direction (`expdir`)

In Bidon's protocol layer (`mediation.proto:121-127`), these map to the
`AdFormat` enum:

| Format      | Description                  | Typical Dimensions |
|-------------|------------------------------|--------------------|
| BANNER      | Standard banner              | 320×50             |
| LEADERBOARD | Wide leaderboard             | 728×90             |
| MREC        | Medium rectangle             | 300×250            |
| ADAPTIVE    | Device/screen-adaptive width | Auto               |

Bidon extends IAB's `DisplayPlacement` via `DisplayPlacementExt`
(`mediation.proto:129-136`) to carry:

- `format` (AdFormat enum)
- `orientation` (portrait/landscape)

## Mapping: Ad Type → OpenRTB Bid Request

Each Bidon ad type maps to a specific OpenRTB `Imp` object structure when
bidding adapters construct bid requests for demand sources.
Each adapter constructs the `Imp` differently and this table is only a rough guide:

| Bidon Ad Type   | OpenRTB Imp Object  | Key Fields                                 |
|-----------------|---------------------|--------------------------------------------|
| Banner + Format | Banner              | `banner.w/h` from format, `instl:0`        |
| Interstitial    | Banner (fullscreen) | `banner` with fullscreen dims, `instl:1`   |
| Rewarded        | Video               | `video` with rewarded extension, `instl:1` |

Interstitial and Rewarded ad types do not carry an Ad Format — the format
is implicit (fullscreen).

For Banner ad types, the format is expanded for cross-compatibility:

- `ADAPTIVE` → also matches `LEADERBOARD` (tablet) or `BANNER` (phone)
- `BANNER` / `LEADERBOARD` ↔ also matches `ADAPTIVE` (backward compatibility)

See `internal/auction/store/ad_units_matcher.go:selectAdFormats`.

## System Flow

1. **App/SDK** → sends auction request with `ad_type` (`banner`, `interstitial`,
   `rewarded`) and optional `ad_object` carrying format/creative details.

2. **SDK API** → matches auction configuration by `ad_type` from the DB
   (`auction_configurations` table).

3. **Auction Service** → filters line items by `ad_type` + `format`
   (with format expansion for cross-compatibility).

4. **Bidding Adapters** → translate Bidon `ad_type` into OpenRTB `Imp` impression objects
   for each demand source.

5. **Bid Response** → follows the same mapping back to the app/SDK.

## References

- IAB AdCOM proto: `proto/adcom/proto/com/iabtechlab/adcom/v1/`
- Bidon mediation proto: `proto/proto/org/bidon/proto/v1/mediation/mediation.proto`
- Internal ad type definitions: `internal/ad/ad.go` (`ad.Type`, `ad.Format`)
- DB ad type mapping: `internal/db/db.go` (`AdTypeFromDomain`)
- Ad units matching: `internal/auction/store/ad_units_matcher.go`
- OpenRTB spec: <https://github.com/prebid/openrtb>
- Bidding adapters: `internal/bidding/adapters/`
