# Reference: Ad Formats, Creative Types, and Platform Notes

Supporting detail for [SKILL.md](SKILL.md). Read this when a request
involves a specific ad format, creative type, or the store-kit install
overlay, and the compact tables in SKILL.md aren't enough context to make
a correct pairing.

Schema validity and runtime effect are different concerns. Soft
applicability below must **not** be turned into server-side validation
failures — Ineffective combinations remain legal JSON; agents should
counsel and omit by default (see Agent policy in SKILL.md).

## Ad formats (`container.format`)

| Format         | Typical size / behavior                        | Rendering-config notes                                              |
|----------------|--------------------------------------------------|-----------------------------------------------------------------------|
| banner         | Fixed-size (320x50, 300x250, 728x90, adaptive)   | Auto-refresh, sticky positioning, collapse on empty. `close_button` / `endcards` / playback fields are usually no-ops (banners aren't skip-dismissible; static images don't mute). Prefer `impression_tracking` and a matching `creative.type`. |
| interstitial   | Full-screen overlay                              | Pair with `close_button.delay_seconds` to control skip timing and `container.orientation` to lock rotation. Endcards and video playback knobs are in scope when the creative supports them. |
| rewarded       | Full-screen with completion reward               | Set `container.skip_offset_seconds` (time before skip is allowed) and usually `close_button.visible: false` until reward completion — the platform SDK, not this config, enforces the reward gate. |
| native         | Publisher-templated structured assets            | Use `creative.type: "native"` (OpenRTB Native 1.2 assets, not this JSON schema, carries the actual asset payload). Most `container`/`close_button` fields don't apply — the publisher's template controls layout. |
| mrec           | Inline 300x250 within content                    | `container.impression_tracking` matters most here (viewability-driven billing). MRAID expand is common — pair with `creative.type: "mraid"`. Same chrome/playback caveats as banner for static creatives. |

## Format × field applicability

Use this when mapping a user's requested knobs onto a config. **Effective**
means the uSDK typically honors the field for that format; **no-op** means
the value is accepted but does not change rendering for the usual creative
on that format; **optional** means it can matter when the rest of the
setup supports it (e.g. video in an MREC, or store kit with a real app id).

| Field / section | banner | mrec | interstitial | rewarded | native |
|-----------------|--------|------|--------------|----------|--------|
| `close_button.*` | no-op (usual) | no-op (usual) | effective | effective (often gated) | no-op (usual) |
| `endcards.*` | no-op (usual) | no-op (usual) | effective | effective | no-op (usual) |
| `container.format` | set explicitly | set explicitly | set explicitly | set explicitly | set explicitly |
| `container.orientation` | optional | optional | effective | effective | no-op (usual) |
| `container.background_*` | optional | optional | effective | effective | no-op (usual) |
| `mute_on_start` | no-op unless video | no-op unless video | video only | video only | no-op |
| `max_duration_seconds` | no-op unless video | no-op unless video | video only | video only | no-op |
| `skip_offset_seconds` | no-op | no-op | optional | effective | no-op |
| `impression_tracking` | effective | effective | optional | optional | optional |
| `creative.type` / versions | must match markup | must match markup | must match markup | must match markup | `native` |
| `creative.preload_strategy` | optional | optional | optional | optional | optional |
| `store_kit.*` | optional | optional | optional | optional | optional |

**Creative-type overlays** (apply on top of the format row):

| Creative type | Playback fields (`mute_on_start`, duration, skip) | Notes |
|---------------|-----------------------------------------------------|--------|
| `static_image` | no-op | Default if `creative.type` omitted — dangerous mismatch for HTML/VAST |
| `html` | no-op unless the markup itself plays media | Common for banners; do not assume video knobs apply |
| `mraid` | depends on the creative | Expand/resize matter more than mute for many MRAIDs |
| `vast` | effective | Real home for mute / duration / skip; no separate `autoplay` field |
| `playable` | usually n/a | Interactive HTML/MRAID semantics |
| `native` | no-op | Publisher template owns presentation |

Agent rule of thumb: **playback fields require a video creative; chrome
fields (close/endcards) require a full-screen container.**

## Creative types (`creative.type`) and how each is rendered

| Creative Type | Standards                         | Rendering approach                                                                             |
|----------------|-------------------------------------|--------------------------------------------------------------------------------------------------|
| mraid          | MRAID 2.0, 3.0                     | WebView with MRAID bridge injection (expand, resize, viewability APIs). Set `creative.mraid_version` to match what the markup was authored against. |
| vast           | VAST 3.0-4.2, VPAID 2.0 (opt-in)   | Native video player parses the VAST XML. Companion ads and OM SDK verification supported. Set `creative.vpaid_enabled: true` only if the DSP's creative actually requires VPAID — it's an extra attack surface, off by default. |
| html           | HTML5, CSS3, JS                    | Sandboxed iframe/WebView. `creative.html_sandbox_policy` sets the iframe `sandbox` attribute — tighten it if the creative doesn't need script execution beyond display. |
| static_image   | JPEG, PNG, GIF, WebP                | Native image rendering, aspect-fit/fill, click-through overlay. This is the fallback default if `creative.type` is left unset — see the warning in SKILL.md. |
| native         | OpenRTB Native 1.2                  | Structured asset delivery (title, icon, image, CTA, rating) rendered into the publisher's template. |
| playable       | MRAID 3.0 + HTML5                   | Interactive HTML5 in a sandboxed WebView with gesture passthrough. Functionally an HTML creative with `mraid_version: "3.0"` semantics. |

A mismatch between `creative.type` and the actual markup in
`seatbid.bid.adm`/`nurl` does not raise an error — the renderer just
applies the wrong rendering strategy to the markup. Always set this
field to match reality rather than relying on the default.

## Endcard asset hosting

DSPs have two options for `endcards.assets[].url`:

1. **DSP-hosted (default path)**: host the images/assets on your own CDN
   and reference those URLs directly. The uSDK fetches them at render
   time.
2. **Bidon-hosted**: if you don't have your own asset-hosting
   infrastructure, Bidon provides a creative asset upload API and
   dashboard UI. Upload assets ahead of time, get back a Bidon-hosted CDN
   URL, and reference that URL the same way.

Both options use the identical `endcards.assets[]` schema — the uSDK
does not distinguish between the two. Pick whichever gets you consistent
CDN performance with the least new infrastructure.

## Store Kit platform notes (native in-app install)

| Platform | Feature                | Notes                                                                                                       |
|----------|--------------------------|----------------------------------------------------------------------------------------------------------------|
| iOS      | SKOverlay               | Bottom overlay or full "overlay" presentation (`store_kit.position: "overlay"`), iOS 14+. Supports app-clip and full-app presentations. |
| Android  | Google Inline Installs  | Triggered via Google Play's In-App Review/Install API. `store_kit.position` doesn't change Android's inline presentation — it's an iOS-only distinction. |

`store_kit.trigger` (on_impression / on_endcard / on_click) and
`store_kit.delay_seconds` control *when* the overlay appears relative to
ad lifecycle events, and are honored on both platforms identically. The
uSDK handles the platform API calls and impression/click attribution for
installs initiated through the overlay — nothing further is needed from
the DSP beyond setting these fields correctly.
