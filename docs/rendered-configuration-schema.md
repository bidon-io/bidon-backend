# Rendering Configuration Schema

Each DSP controls how their creatives are rendered by including a rendering configuration JSON in their OpenRTB bid response (e.g., within `seatbid.bid.ext.rendering`). The uSDK Renderer reads this config at render time. All fields are optional with sensible defaults for backward compatibility.

## Close Button Configuration

| Field                          | Type    | Description                                    | Default   |
|--------------------------------|---------|------------------------------------------------|-----------|
| close_button.visible           | boolean | Whether the close button is shown              | true      |
| close_button.delay_seconds     | number  | Seconds before close button appears            | 5         |
| close_button.countdown_visible | boolean | Show countdown timer before close appears      | true      |
| close_button.style             | enum    | icon_x, icon_circle, text_close, custom        | icon_x    |
| close_button.size_dp           | number  | Size in density-independent pixels             | 30        |
| close_button.position          | enum    | top_left, top_right, bottom_left, bottom_right | top_right |
| close_button.color             | string  | Hex color (e.g. #FFFFFF)                       | #FFFFFF   |
| close_button.opacity           | number  | Opacity 0.0–1.0                                | 0.9       |
| close_button.padding_dp        | number  | Padding from screen edges in dp                | 12        |
| close_button.custom_asset_url  | string  | URL to custom icon (when style = custom)       | null      |

## Endcard Configuration

| Field                              | Type    | Description                                                                                                                                       | Default     |
|------------------------------------|---------|---------------------------------------------------------------------------------------------------------------------------------------------------|-------------|
| endcards.enabled                   | boolean | Show endcards after creative                                                                                                                      | false       |
| endcards.count                     | number  | Number of endcards (1–3)                                                                                                                          | 1           |
| endcards.layout                    | enum    | single, carousel, grid                                                                                                                            | single      |
| endcards.auto_advance_seconds      | number  | Carousel auto-advance timing (0 = off)                                                                                                            | 0           |
| endcards.cta_text                  | string  | Call-to-action button text                                                                                                                        | Install Now |
| endcards.cta_color                 | string  | CTA button background color                                                                                                                       | #007AFF     |
| endcards.assets[]                  | array   | Ordered list of endcard asset URLs (`{ "url": "..." }` per entry). URLs can point to DSP-hosted CDN or BidOn-hosted CDN (see asset hosting below) | []          |
| endcards.dismiss_on_background_tap | boolean | Dismiss endcard on background tap                                                                                                                 | true        |

## Endcard Asset Hosting

DSPs have two options for hosting endcard assets:

1. **DSP-hosted (default):** The DSP hosts endcard images/assets on their own CDN and passes the URLs in `endcards.assets[]`. The uSDK fetches assets at render time from the DSP's CDN.
2. **BidOn-hosted:** For DSPs that do not have their own asset-hosting infrastructure, BidOn provides a creative asset upload API (and dashboard UI). DSPs upload endcard assets ahead of time, receive a BidOn-hosted CDN URL, and reference that URL in their bid response. BidOn handles hosting, caching, and low-latency delivery.

Both options use the same `endcards.assets[]` schema — the uSDK does not distinguish between DSP-hosted and BidOn-hosted URLs. The BidOn-hosted option lowers the integration bar for smaller or performance-focused DSPs that are accustomed to uploading creatives to a network's platform, and ensures consistent CDN performance regardless of the DSP's own infrastructure.

## Creative Container and Behavior

| Field                                              | Type    | Description                                  | Default      |
|----------------------------------------------------|---------|----------------------------------------------|--------------|
| container.format                                   | enum    | banner, interstitial, rewarded, native, mrec | interstitial |
| container.orientation                              | enum    | portrait, landscape, responsive              | responsive   |
| container.background_color                         | string  | Background behind the creative               | #000000      |
| container.background_blur                          | boolean | Blur effect behind creative                  | false        |
| container.max_duration_seconds                     | number  | Max playback/display time (0 = unlimited)    | 30           |
| container.skip_offset_seconds                      | number  | Seconds before skip available (rewarded)     | 15           |
| container.mute_on_start                            | boolean | Start video/audio muted                      | true         |
| container.impression_tracking.min_viewable_pct     | number  | % visible for impression                     | 50           |
| container.impression_tracking.min_viewable_seconds | number  | Seconds visible for impression               | 1            |

## Creative Type Handling

| Field                        | Type    | Description                                        | Default             |
|------------------------------|---------|----------------------------------------------------|---------------------|
| creative.type                | enum    | mraid, vast, html, static_image, native, playable  | static_image        |
| creative.source              | string  | OpenRTB field: seatbid.bid.adm or seatbid.bid.nurl | (from bid response) |
| creative.mraid_version       | string  | MRAID version (2.0, 3.0)                           | 3.0                 |
| creative.vast_version        | string  | VAST version (3.0, 4.0, 4.1, 4.2)                  | 4.2                 |
| creative.vpaid_enabled       | boolean | Allow VPAID inside VAST                            | false               |
| creative.html_sandbox_policy | string  | Sandbox attribute for HTML iframe                  | allow-scripts       |
| creative.preload_strategy    | enum    | eager, lazy, on_demand                             | eager               |

## In-App Store Kit Configuration

| Field                      | Type    | Description                                                                                                                                                         | Default               |
|----------------------------|---------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------------------|
| store_kit.enabled          | boolean | Enable in-app install overlay (SKOverlay on iOS, Inline Install on Android)                                                                                         | false                 |
| store_kit.app_store_id     | string  | App Store ID (iOS) or package name (Android) for the advertised app                                                                                                 | (required if enabled) |
| store_kit.trigger          | enum    | on_impression, on_endcard, on_click                                                                                                                                 | on_endcard            |
| store_kit.delay_seconds    | number  | Delay in seconds after the trigger event before the store overlay is rendered. Allows the creative to display for a set duration before showing the install prompt. | 0                     |
| store_kit.user_dismissable | boolean | Whether the user can dismiss the store overlay (e.g., swipe away or tap close). When false, the overlay persists until the ad is closed or the user taps Install.   | false                 |
| store_kit.dismiss_on_close | boolean | Dismiss the store overlay when the parent ad is closed                                                                                                              | true                  |
| store_kit.position         | enum    | bottom, overlay (iOS SKOverlay only)                                                                                                                                | bottom                |
| store_kit.campaign_token   | string  | Campaign attribution token for install tracking                                                                                                                     | null                  |

## Supported Ad Formats and Creative Types

### Ad Formats (Container Types)

| Format         | Description                                    | Key Behaviors                                       |
|----------------|------------------------------------------------|-----------------------------------------------------|
| Banner         | Fixed-size (320x50, 300x250, 728x90, adaptive) | Auto-refresh, sticky positioning, collapse on empty |
| Interstitial   | Full-screen overlay                            | Configurable close-button delay, orientation lock   |
| Rewarded Video | Full-screen with completion reward             | Skip offset, completion callback, reward validation |
| Native         | Publisher-templated structured assets          | Asset binding, click regions, privacy icon          |
| MREC           | Inline 300x250 within content                  | Viewability tracking, MRAID expand support          |

### Creative Types (from OpenRTB Bid Response)

| Creative Type | Standards                        | Rendering Approach                                                                             |
|---------------|----------------------------------|------------------------------------------------------------------------------------------------|
| MRAID         | MRAID 2.0, 3.0                   | WebView with MRAID bridge injection. Expand, resize, viewability APIs.                         |
| VAST / VPAID  | VAST 3.0–4.2, VPAID 2.0 (opt-in) | Native video player with VAST XML parsing. Companion ads, verification (OM SDK). VPAID opt-in. |
| HTML          | HTML5, CSS3, JS                  | Sandboxed iframe/WebView. DSP controls sandbox policy via rendering config.                    |
| Static Image  | JPEG, PNG, GIF, WebP             | Native image rendering with aspect-fit/fill. Click-through overlay.                            |
| Native Assets | OpenRTB Native 1.2               | Structured asset delivery (title, icon, image, CTA, rating). Publisher template.               |
| Playable      | MRAID 3.0 + HTML5                | Interactive HTML5 in sandboxed WebView. Gesture passthrough.                                   |

## In-App Store Kit Support

The uSDK supports native in-app install experiences on both iOS and Android, enabling users to install advertised apps without leaving the publisher's app:

| Platform | Feature                | Description                                                                                                                                                                                                                                                                 |
|----------|------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| iOS      | SKOverlay              | Renders an App Store overlay at the bottom of the screen during or after ad display. Supports both app-clip and full-app presentations. Configurable via the DSP's rendering config (show on impression, on endcard, or on click). Uses StoreKit's SKOverlay API (iOS 14+). |
| Android  | Google Inline Installs | Triggers an in-app install flow via Google Play's In-App Review / Install API, allowing users to install the advertised app without navigating away. Configurable trigger (on impression, on endcard, or on click) via rendering config.                                    |

Both features are activated via the DSP's rendering configuration JSON in the bid response. The uSDK handles the platform-specific API calls and provides impression/click attribution for installs initiated through these flows.

## Defaulting and Validation Rules

These rules determine what actually reaches the uSDK:

1. **Every section is always backfilled with its documented defaults.** Whether you omit `rendering` entirely, send `"rendering": {}`, or omit a specific section (e.g. no `close_button` key at all), the uSDK receives a fully populated section either way — there's no separate opt-in step. Only set the fields you actually want to override; everything else, including whole sections, defaults for you.
2. **A section with an invalid field falls back to that section's defaults as a whole**, not the whole config. Other, valid sections you sent are preserved as-is. This is section-level, not field-level, because fields within a section are often connected (e.g. `store_kit.app_store_id` is only required *because* `store_kit.enabled` is `true`; `close_button.custom_asset_url` is only meaningful *because* `close_button.style` is `custom`). A failure on one field can be a symptom of — or trigger — related failures on the fields connected to it, so patching only the field that happened to fail could still leave the section in a combination that was never valid together. Resetting the whole section guarantees it lands on a combination that is internally consistent. The one exception is `endcards.assets[]`: each array entry is validated independently of the others and of the rest of the section, so an individual asset with a bad `url` is dropped from the array, while the rest of the endcards config (and the other valid assets) is kept.
3. Every rejected section is logged server-side (with the DSP's demand ID and which section failed) so misconfigurations are discoverable. Nothing is ever returned to the SDK to indicate a section was rejected, and OpenRTB has no mechanism to reject the bid itself. If your close button isn't behaving as configured, check with Bidon whether that section is being rejected.

## Examples

Each `rendering` object below is what a DSP places at `seatbid.bid.ext.rendering` in its OpenRTB bid response.

### Example 1 — Interstitial with a custom close button

Top-left custom icon, no countdown, appears after 3 seconds. `style: "custom"` requires `custom_asset_url` — both are set together. `creative.type` is set explicitly rather than left to default.

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

### Example 2 — Endcard carousel with BidOn-hosted assets

`auto_advance_seconds` only takes effect because `layout` is `"carousel"`. `count: 3` matches the number of `assets` entries — the schema does not cross-check this, so keep them in sync yourself.

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

### Example 3 — Invalid field causes a section-level fallback

`"on_open"` is not a valid `store_kit.trigger` value (must be `on_impression`, `on_endcard`, or `on_click`). `close_button` validates cleanly and is preserved. `store_kit` reverts to its defaults entirely — including `enabled: false` — because one field in it was invalid. The store overlay silently never shows, and nothing in the OpenRTB response indicates why.

**Sent:**

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

**What reaches the uSDK** (`endcards`, `container`, and `creative` are also present and fully defaulted; omitted here for clarity):

```json
{
  "close_button": {
    "style": "icon_circle",
    "color": "#112233",
    "...": "rest defaulted"
  },
  "store_kit": {
    "enabled": false,
    "...": "rest defaulted — the whole section, not just trigger"
  }
}
```
