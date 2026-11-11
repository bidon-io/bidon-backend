# ADR-0005: Standalone admin UI

**Status:** Accepted
**Date:** 2026-06-25
**Linear:** [BAC-16](https://linear.app/bidon/issue/BAC-16/deploy-admin-ui-as-standalone-nuxt-container-with-same-origin-api-and), completed by [BAC-20](https://linear.app/bidon/issue/BAC-20/remove-embedded-static-ui-from-bidon-admin-after-nuxt-container)

---

## Context

The admin UI was a Nuxt app generated to static files and embedded in the `bidon-admin` Go binary with `go:embed`. Every UI change rebuilt the API image. Serving the UI from another origin would put the session cookie on a different host and force CORS on `/api` and `/auth`.

## Decision

`bidon-ui` is its own Nuxt container. The browser talks only to that origin. Nitro middleware proxies `/api/**` and `/auth/**` to `bidon-admin` (`NUXT_API_PROXY_TARGET`). The frontend calls relative `/api` and `/auth` paths.

`bidon-admin` is API-only. It does not embed or serve the admin HTML. `/docs` on the admin port stays; that is the OpenAPI spec, not the UI.

Rejected: keep the static bundle inside the Go binary. Also rejected: an nginx or Spaces front that calls the API on another origin.

## Consequences

- Browser traffic has to hit `bidon-ui` (local port 3010, container port 3000). `http://localhost:1323/` is the API and will not render the admin.
- Session login keeps working because the cookie is same-origin. The web client still sends `X-Bidon-App: web`. Programmatic API-key and Bearer clients are unchanged.
- Coolify previews resolve the admin through `SERVICE_NAME_BIDON_ADMIN` when that is set. UI and API images are built and tagged separately, so a new admin image does not update the UI until `BIDON_UI_TAG` moves too.
- The admin image has no Node build stage.
