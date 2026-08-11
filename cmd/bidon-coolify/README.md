# bidon-coolify

CLI utility for automating Bidon app setup in Coolify via API.

## Required auth inputs

- `COOLIFY_BASE_URL` (or `--base-url`)
- `COOLIFY_API_KEY` (or `--api-key`)

## Commands

### 1) Configure GitHub repo integration

Registers a GitHub App integration in Coolify:

```bash
go run ./cmd/bidon-coolify configure-github-repo \
  --name bidon-gh-app \
  --organization bidon-io \
  --app-id 123456 \
  --installation-id 654321 \
  --client-id iv1.xxxxx \
  --client-secret xxxxx \
  --webhook-secret xxxxx \
  --private-key-uuid <coolify_private_key_uuid>
```

### 2) Create app from configured repo

Creates an app using a private repo integration. Defaults are tuned for prebuilt-image deployment (`--build-pack=dockerimage`, `--use-build-server=false`):

```bash
go run ./cmd/bidon-coolify create-app \
  --project-uuid <project_uuid> \
  --server-uuid <server_uuid> \
  --github-app-uuid <github_app_uuid> \
  --name bidon-admin \
  --git-repository https://github.com/bidon-io/bidon-backend \
  --git-branch main \
  --image-name ghcr.io/bidon-io/bidon-admin \
  --image-tag latest \
  --ports-exposes 1323 \
  --health-check-enabled \
  --health-check-path /health \
  --health-check-port 1323
```

For public repositories, omit `--github-app-uuid`.

### 3) Configure app environment variables

Inline:

```bash
go run ./cmd/bidon-coolify configure-app-env \
  --app-uuid <application_uuid> \
  --env STAGING_DATABASE_URL=postgres://... \
  --env REDIS_URL=redis://...
```

From file:

```bash
go run ./cmd/bidon-coolify configure-app-env \
  --app-uuid <application_uuid> \
  --env-file .env.coolify
```

You can combine `--env` and `--env-file`; inline values are merged with file values.
