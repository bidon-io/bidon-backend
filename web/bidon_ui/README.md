# Bidon Admin UI (Nuxt 3)

Look at the [Nuxt 3 documentation](https://nuxt.com/docs/getting-started/introduction) to learn more.

## Setup

Make sure to install the dependencies:

```bash
yarn install
```

## Development Server

Start the development server on `http://localhost:3000`:

```bash
yarn dev
```

The Nitro middleware (`server/middleware/proxy.ts`) proxies `/api/**` and `/auth/**` to the Go `bidon-admin` backend (`NUXT_API_PROXY_TARGET`, defaults to `http://localhost:1323`).

With the full local stack (`docker compose -f docker-compose.dev.yml up`), the UI is available at `http://localhost:3010`.

## Local API

```bash
go run ./cmd/bidon-admin
```

## Production build

The UI runs as a standalone Nuxt/Node.js container (`bidon-ui`).

```bash
# Build (outputs Nitro server + static assets to .output/)
yarn build

# Run the production server
NUXT_API_PROXY_TARGET=http://localhost:1323 node .output/server/index.mjs
# → http://localhost:3000
```

**Required environment variable:**
- `NUXT_API_PROXY_TARGET` — URL of the Go `bidon-admin` backend (e.g. `http://bidon-admin:1323`)

**Docker:** build the `bidon-ui` target in the root `Dockerfile` (`just build-ui` / `just ci-build-ui`).

Check out the [deployment documentation](https://nuxt.com/docs/getting-started/deployment) for more information.
