# ADR-0009: Staging images and preview databases

**Status:** Accepted
**Date:** 2026-08-11
**Linear:** [BAC-14](https://linear.app/bidon/issue/BAC-14/build-and-deploy-staging-services-from-do-container-registry-images), [BAC-52](https://linear.app/bidon/issue/BAC-52/add-postgres-service-to-staging-docker-compose-for-coolify-staging-and)

---

## Context

Coolify was building staging from the checkout it had, which drifted from the image a developer had just tested and made preview builds depend on Coolify's builder. Previews also inherited a baked `DATABASE_URL`. Coolify expands `${DATABASE_URL:-...}` to the production value, so a preview could attach to the staging or production database.

## Decision

Staging and preview services run prebuilt images from `registry.digitalocean.com/bidon-io/<service>:<sha>`. `just ci-build-*` pushes them. Coolify selects the running bits with `BIDON_ADMIN_TAG`, `BIDON_SDKAPI_TAG`, `BIDON_MIGRATE_TAG`, `BIDON_SEED_TAG`, and `BIDON_UI_TAG`. A deploy does not change until those tags change.

`docker-compose.staging.yml` always starts a Postgres container. Apps connect there unless `STAGING_DATABASE_URL` is set. That variable, not `DATABASE_URL`, is the override, so a preview does not pick up the production URL Coolify would otherwise interpolate. Migrate waits until Postgres is healthy.

How to build and point Coolify at a SHA is the [staging runbook](../runbooks/staging.md).

Rejected: letting Coolify build from the git tree on each deploy. Also rejected: one shared staging database for every preview.

## Consequences

- Backend, seed, and UI tags move independently. A new admin image does not update the UI until `BIDON_UI_TAG` is the same SHA.
- Local `docker-compose.dev.yml` still mounts source and hot-reloads `bidon-ui`. Staging will not show a working-tree change that was never pushed as an image.
- The preview Postgres container runs even when `STAGING_DATABASE_URL` points the apps at a managed database. The container is not the data, the URL is.
