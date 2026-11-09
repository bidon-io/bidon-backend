# Staging and images

Commands below run from the repository root.

Images are tagged with the current git SHA and named `registry.digitalocean.com/bidon-io/<service>:<sha>`.

## Local builds

Build images into your local Docker store for `docker run` or compose testing. No registry login or push:

```shell
just build-admin    # bidon-admin
just build-sdkapi   # bidon-sdkapi
just build-migrate  # bidon-migrate
just build-seed     # bidon-seed
just build-dspsim   # bidon-dspsim
just build-ui       # bidon-ui
just build-telemetry  # bidon-telemetry
just build-all      # all of the above
```

## CI / registry builds

Build and push to the DigitalOcean Container Registry. Authenticate first:

```shell
just docker-login
```

Push a single service:

```shell
just ci-build-admin
just ci-build-sdkapi
just ci-build-migrate
just ci-build-seed
just ci-build-dspsim
just ci-build-ui
just ci-build-telemetry
```

Or push all services (logs in once):

```shell
just ci-build-all
```

## Staging deployment

Staging runs in [Coolify](https://coolify.bidon.squads.com/).

After pushing images with `just ci-build-*`, update the app configuration there. Set each `BIDON_*_TAG` environment variable to the git SHA produced by the build. See [`.env.staging.example`](../../.env.staging.example) for the full list of required variables.

**Check image versions in Coolify.** Staging and production run pre-built registry images, not your local source tree. Unlike `docker-compose.dev.yml` (which mounts live code into `bidon-ui`), Coolify only changes what you deploy when you push new images and update the tag env vars. Backend, seed, and UI images move independently. After feature work that touches both backend and admin UI, rebuild and redeploy all affected services (often including `just ci-build-ui`) and set every `BIDON_*_TAG` in Coolify to the same git SHA before assuming staging matches `new-main`.

For initial Coolify setup (Terraform provisioning, `bidon-coolify` CLI), see [infra/README.md](../../infra/README.md).
