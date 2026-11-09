# ADR-0007: Rendering config placement

**Status:** Accepted
**Date:** 2026-09-23
**Linear:** [BAC-19](https://linear.app/bidon/issue/BAC-19/expose-dsp-controlled-rendering-configuration-from-openrtb-bid), [BAC-70](https://linear.app/bidon/issue/BAC-70/nest-auction-rendering-under-extrendering-and-leave-creative-type)

---

## Context

A DSP can tell the uSDK how to render its creative: close button, endcards, container. That JSON arrives on the OpenRTB bid at `seatbid.bid.ext.rendering`. The auction has to hand it to the SDK.

Shipped SDK builds only forward the auction ad unit's `ext` object to the renderer. A sibling field next to `ext` never reaches them. The first cut of this work put rendering beside `ext`, which those builds dropped.

The server also used to fill `creative.type` with `static_image` when the DSP omitted it. A video or HTML markup then rendered as a static image.

## Decision

The auction copies the DSP block to `ad_units[].ext.rendering` on REST. On gRPC it is the `rendering` entry of the `BidExt.ext` map, as a JSON string. `BidExt.rendering` stays unset, so there is one path.

When the DSP omits `creative.type`, the server leaves it empty. The SDK detects video, HTML, or MRAID from the markup. A DSP that sets a type, such as `vast`, still gets that value through.

Invalid sections fall back to that section's defaults. The other sections are kept. The field list itself is the [rendering schema](../architecture/rendered-configuration-schema.md).

Rejected: a top-level `rendering` field, which would need a new SDK release before any DSP config worked. Also rejected: defaulting a missing creative type to `static_image`.

## Consequences

- A DSP config only takes effect if it is inside `ext`. Clients that read a sibling `rendering` field will not see it.
- An empty `creative.type` is deliberate. Filling it in "to be helpful" makes video and HTML creatives render as images.
- Amazon and any adapter that does not parse `bid.Ext` still get a default config. The builder derives it in one place from the raw extension.
