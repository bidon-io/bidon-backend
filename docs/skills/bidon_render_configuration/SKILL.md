---
name: bidon_render_configuration
metadata.version: "0.0.1"
description: >-
  Author and validate the JSON rendering configuration DSPs place at
  seatbid.bid.ext.rendering in an OpenRTB bid response to the Bidon ad
  mediation platform, controlling how the Bidon uSDK renders a DSP's
  creative (close button, endcards, container/format, creative-type
  handling, in-app store overlays). Use when writing, reviewing, or
  debugging a Bidon rendering config; when a user asks "how do I
  configure the close button / endcard / store overlay for Bidon"; or
  when a Bidon integration's creative isn't rendering as expected and
  the cause may be a malformed rendering config.
---

# Bidon Render Configuration

Bidon is an ad-mediation platform. DSPs (demand-side platforms) that win a
bid can control how their creative is rendered by the Bidon uSDK by
including a `rendering` object in their OpenRTB bid response, at
`seatbid.bid.ext.rendering`. Only DSPs set this — publishers never touch
it. Every field is optional and defaults sensibly, but a single invalid
field can silently take an entire section back to its defaults with no
error reported anywhere. Read "Defaulting rules" before writing any
config.

## Defaulting rules (read this first)

These rules decide what actually reaches the renderer.

1. **Every section is always backfilled with its documented defaults.**
   Omitting `rendering` entirely, sending `"rendering": {}`, and omitting
   one specific section (e.g. no `close_button` key) all produce the same
   result for that section: a fully populated object with the documented
   defaults. There is no separate opt-in step — set only the fields you
   want to override, on only the sections you want to override; leave
   everything else out.
2. This is recursive: a nested optional object (e.g.
   `container.impression_tracking`) is backfilled the same way whether
   its parent section is present, empty, or omitted.
3. **A section with one invalid field falls back to that section's
   defaults as a whole** — not the whole config, and not just the bad
   field. This is section-level rather than field-level because fields
   within a section are often connected: `store_kit.app_store_id` is only
   required *because* `store_kit.enabled` is `true`; `close_button.custom_asset_url`
   is only meaningful *because* `close_button.style` is `custom`. A
   failure on one field can be a symptom of, or trigger, related failures
   on the fields it's connected to, so patching only the field that
   happened to fail could still leave the section in a combination that
   was never valid together — resetting the whole section guarantees it
   lands on a combination that's internally consistent. Other sections
   you configured correctly are preserved exactly as sent regardless. The
   one field-level exception is `endcards.assets[]`: each entry validates
   independently of the others and of the rest of the section, so a
   single bad asset URL is dropped from the array; the rest of the
   endcards config (and the other valid assets) survives.
4. **There is no feedback loop.** OpenRTB has no mechanism for Bidon to
   reject a bid because of a bad rendering config, and no validation
   error is ever surfaced to the DSP. A malformed section is logged
   server-side on Bidon's end and then silently replaced with defaults.
   **Self-validate before sending** — treat every rule and enum below as
   load-bearing, because there is no runtime signal telling you when
   you've broken one.

## Workflow

1. Identify which of the five top-level sections the request actually
   wants to customize: `close_button`, `endcards`, `container`,
   `creative`, `store_kit`. Only include a section if you're overriding
   at least one of its fields — omitting it entirely gets you the same
   defaults as sending `{}`, so there's never a reason to send an empty
   section on purpose.
2. Within a section you do include, set only the fields the request is
   overriding; everything else in that section still defaults.
3. Fill in fields using the tables below. Check "Cross-field dependencies"
   for any field you touch that has one.
4. Run the self-check list before finalizing.
5. Emit only `seatbid.bid.ext.rendering` — do not wrap it in anything
   else.

See [examples.md](examples.md) for complete worked payloads, including a
before/after pair showing rule 3 in action.

## Close Button

| Field                          | Type    | Values / Constraint                            | Default   |
|---------------------------------|---------|------------------------------------------------|-----------|
| close_button.visible           | boolean |                                                  | true      |
| close_button.delay_seconds     | number  | >= 0                                            | 5         |
| close_button.countdown_visible | boolean |                                                  | true      |
| close_button.style             | enum    | icon_x, icon_circle, text_close, custom         | icon_x    |
| close_button.size_dp           | number  | >= 0                                            | 30        |
| close_button.position           | enum    | top_left, top_right, bottom_left, bottom_right  | top_right |
| close_button.color             | string  | hex color, `#RGB` or `#RRGGBB`                  | #FFFFFF   |
| close_button.opacity           | number  | 0.0 - 1.0                                       | 0.9       |
| close_button.padding_dp        | number  | >= 0                                            | 12        |
| close_button.custom_asset_url  | string  | absolute http(s) URL                            | null      |

**Dependency**: `custom_asset_url` is only required/validated as a URL
when `style == "custom"`. If you set `style: "custom"` you **must** also
set a valid `custom_asset_url`, or the entire `close_button` section is
rejected and falls back to defaults (which is `style: icon_x`, silently
discarding the "custom" intent).

## Endcards

| Field                              | Type    | Values / Constraint                     | Default     |
|-------------------------------------|---------|------------------------------------------|-------------|
| endcards.enabled                   | boolean |                                          | false       |
| endcards.count                     | number  | 1 - 3                                    | 1           |
| endcards.layout                    | enum    | single, carousel, grid                   | single      |
| endcards.auto_advance_seconds      | number  | >= 0 (0 = off, relevant for carousel)    | 0           |
| endcards.cta_text                  | string  |                                          | "Install Now" |
| endcards.cta_color                 | string  | hex color, `#RGB` or `#RRGGBB`          | #007AFF     |
| endcards.assets[]                  | array   | `[{ "url": "https://..." }, ...]`       | []          |
| endcards.dismiss_on_background_tap | boolean |                                          | true        |

**Dependency**: `assets[].url` must be an absolute http(s) URL — anything
else drops just that entry (see rule 3). `auto_advance_seconds` only has
an effect when `layout: "carousel"`. Asset URLs may point to the DSP's
own CDN or to a Bidon-hosted CDN (DSPs without asset infrastructure can
upload via Bidon's creative asset API/dashboard and reference the
returned URL) — the schema is identical either way.

## Container (format & playback behavior)

| Field                                              | Type    | Values / Constraint                          | Default      |
|-----------------------------------------------------|---------|------------------------------------------------|--------------|
| container.format                                   | enum    | banner, interstitial, rewarded, native, mrec | interstitial |
| container.orientation                              | enum    | portrait, landscape, responsive              | responsive   |
| container.background_color                         | string  | hex color                                     | #000000      |
| container.background_blur                          | boolean |                                                | false        |
| container.max_duration_seconds                     | number  | >= 0 (0 = unlimited)                          | 30           |
| container.skip_offset_seconds                      | number  | >= 0 (relevant for rewarded)                  | 15           |
| container.mute_on_start                            | boolean |                                                | true         |
| container.impression_tracking.min_viewable_pct     | number  | 0 - 100                                       | 50           |
| container.impression_tracking.min_viewable_seconds | number  | >= 0                                          | 1            |

`impression_tracking` is a nested optional object — it's backfilled with
its defaults the same as any other section (see defaulting rule 2), so
only include it if you're overriding `min_viewable_pct` or
`min_viewable_seconds`.

## Creative (how the markup is interpreted)

| Field                        | Type    | Values / Constraint                               | Default        |
|-------------------------------|---------|------------------------------------------------------|----------------|
| creative.type                | enum    | mraid, vast, html, static_image, native, playable    | static_image   |
| creative.source              | enum    | seatbid.bid.adm, seatbid.bid.nurl                    | seatbid.bid.adm |
| creative.mraid_version       | enum    | 2.0, 3.0                                              | 3.0            |
| creative.vast_version        | enum    | 3.0, 4.0, 4.1, 4.2                                    | 4.2            |
| creative.vpaid_enabled       | boolean |                                                        | false          |
| creative.html_sandbox_policy | string  | HTML iframe `sandbox` attribute value                 | allow-scripts  |
| creative.preload_strategy    | enum    | eager, lazy, on_demand                                | eager          |

**Always set `creative.type` explicitly.** It defaults to `static_image`
if omitted, which silently mismatches the renderer against your creative
if the markup is actually video/HTML/MRAID/native/playable — there is no
error, just a mis-rendered ad.

## Store Kit (native in-app install overlay: iOS SKOverlay / Android Inline Install)

| Field                      | Type    | Values / Constraint                          | Default   |
|------------------------------|---------|------------------------------------------------|-----------|
| store_kit.enabled          | boolean |                                                | false     |
| store_kit.app_store_id     | string  | App Store ID (iOS) or package name (Android)  | required if `enabled: true` |
| store_kit.trigger          | enum    | on_impression, on_endcard, on_click           | on_endcard |
| store_kit.delay_seconds    | number  | >= 0, delay after trigger                     | 0         |
| store_kit.user_dismissable | boolean |                                                | false     |
| store_kit.dismiss_on_close | boolean | dismiss overlay when parent ad closes         | true      |
| store_kit.position         | enum    | bottom, overlay (overlay = iOS SKOverlay only) | bottom    |
| store_kit.campaign_token   | string  | attribution token                             | null      |

**Dependency**: if `enabled: true`, `app_store_id` is mandatory. Missing
it invalidates the whole section, which falls back to
`enabled: false` — the overlay silently never shows. `position: "overlay"`
only makes sense on iOS; Android Inline Install always renders inline
regardless of this field.

## Self-check before finalizing

- [ ] Every enum value used is from that field's allow-list above (exact
      string match — these are strict `oneof` checks).
- [ ] Every hex color is `#` + 3 or 6 hex digits.
- [ ] Every "seconds"/"padding"/"size" numeric field is >= 0;
      `opacity` and `min_viewable_pct` are within their stated range.
- [ ] If `close_button.style == "custom"`, `custom_asset_url` is a valid
      absolute URL.
- [ ] If `store_kit.enabled == true`, `app_store_id` is set.
- [ ] Every `endcards.assets[].url` is an absolute http(s) URL.
- [ ] `creative.type` is set explicitly and matches what `seatbid.bid.adm`
      / `nurl` actually contains.
- [ ] Sections the request doesn't ask to customize are left out entirely
      rather than sent as `{}` or with guessed values — omission already
      gets full defaults (rule 1).

## More detail

- [reference.md](reference.md) — ad format ↔ creative type pairings,
  endcard asset hosting, store kit platform notes.
- [examples.md](examples.md) — complete worked JSON payloads, including
  a before/after pair demonstrating the section-level fallback rule.
