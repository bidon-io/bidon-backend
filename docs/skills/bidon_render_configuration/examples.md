# Examples

Worked payloads for [SKILL.md](SKILL.md). Each `rendering` object below is
what a DSP places at `seatbid.bid.ext.rendering` in its OpenRTB bid
response.

## 1. Interstitial with a custom close button

Top-left custom icon, no countdown, appears after 3 seconds:

```json
{
  "rendering": {
    "close_button": {
      "style": "custom",
      "custom_asset_url": "https://cdn.example-dsp.com/icons/close.png",
      "position": "top_left",
      "delay_seconds": 3,
      "countdown_visible": false
    },
    "creative": { "type": "html" }
  }
}
```

`style: "custom"` requires `custom_asset_url` — both are set together
(see the close-button dependency in SKILL.md). `creative.type` is set
explicitly rather than left to default.

## 2. Endcard carousel with Bidon-hosted assets

```json
{
  "rendering": {
    "endcards": {
      "enabled": true,
      "count": 3,
      "layout": "carousel",
      "auto_advance_seconds": 4,
      "cta_text": "Get the App",
      "assets": [
        { "url": "https://cdn.bidon.io/creatives/dsp123/endcard1.png" },
        { "url": "https://cdn.bidon.io/creatives/dsp123/endcard2.png" },
        { "url": "https://cdn.bidon.io/creatives/dsp123/endcard3.png" }
      ]
    },
    "creative": { "type": "vast" }
  }
}
```

`auto_advance_seconds` only takes effect because `layout` is
`"carousel"`. `count: 3` matches the number of `assets` entries — the
schema doesn't cross-check this for you, so keep them in sync yourself.

## 3. Rewarded video with a store overlay

```json
{
  "rendering": {
    "container": {
      "format": "rewarded",
      "skip_offset_seconds": 20,
      "mute_on_start": false
    },
    "creative": {
      "type": "vast",
      "vast_version": "4.2"
    },
    "store_kit": {
      "enabled": true,
      "app_store_id": "1234567890",
      "trigger": "on_endcard",
      "delay_seconds": 1
    }
  }
}
```

`app_store_id` is included because `store_kit.enabled` is `true` — omit
either alone and the whole `store_kit` section is rejected server-side
(silently disabled, no error).

## 4. A section you don't mention still shows up, fully defaulted

Sending only `close_button` still gets the renderer a fully populated
`container`, `endcards`, `creative`, and `store_kit` — you don't need to
send `{}` for sections you have no opinion on:

```json
{ "rendering": { "close_button": { "style": "icon_circle" } } }
```

What the renderer actually receives (abridged):

```json
{
  "close_button": { "style": "icon_circle", "...": "rest defaulted" },
  "endcards": { "enabled": false, "...": "all fields defaulted" },
  "container": { "format": "interstitial", "...": "all fields defaulted" },
  "creative": { "type": "static_image", "...": "all fields defaulted" },
  "store_kit": { "enabled": false, "...": "all fields defaulted" }
}
```

There is no way to make a section come back as "absent" once `rendering`
is present at all — omitting a section and sending it as `{}` are the
same thing. If a request only wants to touch `close_button`, the correct
payload is exactly the one-section JSON above; there's no need to also
list the other four sections as empty objects.

## 5. Before / after: one bad field doesn't take down the rest

**Before** (a typo in `store_kit.trigger`):

```json
{
  "rendering": {
    "close_button": { "style": "icon_circle", "color": "#112233" },
    "store_kit": {
      "enabled": true,
      "app_store_id": "1234567890",
      "trigger": "on_open"
    }
  }
}
```

`"on_open"` isn't a valid `store_kit.trigger` value (must be
`on_impression`, `on_endcard`, or `on_click`). What actually reaches the
renderer (`endcards`, `container`, and `creative` are also present and
fully defaulted per example 4, omitted below since this example is about
`close_button` and `store_kit` specifically):

```json
{
  "close_button": { "style": "icon_circle", "color": "#112233", "...": "rest defaulted" },
  "store_kit": { "enabled": false, "...": "rest defaulted — the whole section, not just trigger" }
}
```

`close_button` survives exactly as sent because it validated cleanly.
`store_kit` reverts to its defaults entirely — including `enabled: false`
— because one field in it was invalid. The store overlay silently never
shows, and nothing in the OpenRTB response indicates why.

**After** (fixed):

```json
{
  "rendering": {
    "close_button": { "style": "icon_circle", "color": "#112233" },
    "store_kit": {
      "enabled": true,
      "app_store_id": "1234567890",
      "trigger": "on_endcard"
    }
  }
}
```

## 6. Why the whole section resets, not just the bad field

Fields within a section are often connected, so patching only the field
that happened to fail can't be done safely. `store_kit.app_store_id` is
only required *because* `store_kit.enabled` is `true`:

```json
{ "rendering": { "store_kit": { "enabled": true } } }
```

`app_store_id` is missing, which is invalid whenever `enabled: true`.
What reaches the renderer is `store_kit` reset entirely to defaults —
`enabled: false` included, not just an empty `app_store_id` patched onto
an otherwise-`enabled: true` section:

```json
{ "store_kit": { "enabled": false, "trigger": "on_endcard", "...": "rest defaulted" } }
```

Leaving `enabled: true` while blanking `app_store_id` would produce a
combination — enabled with no app to install — that was never valid in
the first place. The same connection exists between
`close_button.style: "custom"` and `close_button.custom_asset_url`: set
`style: "custom"` without a valid `custom_asset_url` and the whole
`close_button` section resets, not just the URL. `endcards.assets[]` is
the one place this doesn't apply — each entry validates independently of
its siblings, so a bad entry is dropped without touching the rest of the
array or the section.

## 7. Endcards: one bad asset doesn't drop the others

```json
{
  "rendering": {
    "endcards": {
      "cta_text": "Play Now",
      "assets": [
        { "url": "https://cdn.example-dsp.com/good1.png" },
        { "url": "not-a-url" },
        { "url": "https://cdn.example-dsp.com/good2.png" }
      ]
    }
  }
}
```

The renderer receives `endcards.cta_text: "Play Now"` and
`endcards.assets` containing only the two valid entries — this is the one
exception to "one bad field drops the whole section" (see SKILL.md rule 3).

## 8. Correcting ineffective requests on a banner

**User asks** (against an HTML banner / MREC payload — static `<img>` in
markup): yellow close button top-left, video autoplay, carousel auto-advance,
mute on start.

**Infer first:** `container.format: "banner"` (or `"mrec"` for 300x250) and
`creative.type: "html"` from the payload. There is no `autoplay` field.

**Do not silently emit** the shopping list. Challenge the Ineffective tier:

- `close_button` — usually no-op on banner/mrec (not skip-dismissible).
- `endcards` carousel — usually no-op without a full-screen post-creative
  surface; empty `assets` would advance nothing even if it were.
- `mute_on_start` — no-op for static image / non-media HTML.
- "Autoplay" — not in the schema; only meaningful once the creative is
  actually video (`vast`).

**Sensible minimal config** to propose instead:

```json
{
  "rendering": {
    "container": {
      "format": "banner",
      "impression_tracking": {
        "min_viewable_pct": 50,
        "min_viewable_seconds": 1
      }
    },
    "creative": {
      "type": "html"
    }
  }
}
```

If the user still wants the yellow control after the warning, include only
that override (schema-valid, Ineffective) — do not also cargo-cult mute /
carousel / autoplay:

```json
{
  "rendering": {
    "close_button": {
      "position": "top_left",
      "color": "#FFFF00"
    },
    "container": { "format": "banner" },
    "creative": { "type": "html" }
  }
}
```

**Contrast** — the same close / mute / carousel knobs *are* appropriate
when format and creative support them (full-screen + video), e.g.:

```json
{
  "rendering": {
    "close_button": {
      "position": "top_left",
      "color": "#FFFF00"
    },
    "endcards": {
      "enabled": true,
      "layout": "carousel",
      "auto_advance_seconds": 3,
      "assets": [
        { "url": "https://cdn.example-dsp.com/endcard1.png" },
        { "url": "https://cdn.example-dsp.com/endcard2.png" }
      ]
    },
    "container": {
      "format": "interstitial",
      "mute_on_start": true
    },
    "creative": {
      "type": "vast",
      "preload_strategy": "eager"
    }
  }
}
```
