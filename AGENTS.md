# Agent Instructions

Practical guidance for AI agents working in the Bidon backend codebase.

For full architecture, patterns, and code templates, see [CLAUDE.md](CLAUDE.md).

## Project Overview

Bidon is a **Go-based ad mediation and programmatic advertising platform** handling RTB auctions, ad unit management, demand source adapters, and event tracking.

**Key Technologies:**
- Language: Go
- Database: PostgreSQL (with GORM ORM)
- Caching: Redis
- Messaging: Redpanda (Kafka-compatible API)
- APIs: REST (Echo framework) + gRPC
- Protocol Buffers: For API definitions

## Local Development

### Full local stack (recommended)

```bash
docker compose -f docker-compose.dev.yml up -d
```

Runs Postgres, Redis, Redpanda, migrations, seed data, both API services, and the Nuxt frontend.

| Service          | URL                   |
|------------------|-----------------------|
| bidon-ui         | http://localhost:3010 |
| bidon-admin      | http://localhost:1323 |
| bidon-sdkapi     | http://localhost:1324 |
| Postgres         | localhost:5434        |
| Redpanda         | localhost:19092     |
| Redpanda Console | http://localhost:8080 |

### Manual setup (dependencies only)

```bash
make local-init
docker compose up -d
```

Starts Postgres and Redis. Use the full dev stack above for Redpanda.

### Running services directly

```bash
go run ./cmd/bidon-admin
go run ./cmd/bidon-sdkapi
```

### Migrations

```bash
go run ./cmd/bidon-migrate up
```

### Tests

```bash
make test
go test ./internal/auction/...
```

### Staging / production (Coolify)

Local dev (`docker-compose.dev.yml`) mounts source into `bidon-ui` and hot-reloads. **Coolify runs pre-built registry images** tagged via `BIDON_*_TAG` env vars — not the working tree.

When debugging staging-only admin UI issues, **verify image tags in Coolify** match the commit you expect (especially `BIDON_UI_TAG`). Backend/seed can be newer than `bidon-ui` if only some images were rebuilt and redeployed. Rebuild with `just ci-build-ui` (or `just ci-build-all`) and redeploy with aligned tags. See README “Staging deployment”.

## Key Concepts

- **Repository pattern** for all data access (`internal/*/store/*_repo.go`)
- **Service layer** for business logic (`internal/*/service.go`)
- **User scoping** via `ListOwnedByUser()` / `FindOwnedByUser()` for multi-tenancy
- **Event logging** to Redpanda for analytics (auction, impression, click events)

## Code Style

- Follow standard Go conventions; use `golangci-lint`
- Interfaces defined in consuming packages
- Context as first parameter; explicit error handling (no panics in production)
- Generate mocks with `go generate ./...` (moq)

## Docs and communication

- State what **is** in scope and what to do. Do not pad backlog docs, tickets, or replies with what is **not** being done (“does not rework X”, long non-goals that only name unrelated work).
- Put deferred next steps under Follow-ups. Link related tickets briefly without explaining what they exclude.
- See `.cursor/rules/no-negative-scope.mdc`.

## Verification Checklist

- Run targeted tests in changed packages
- Run `make test` when contracts or shared logic change
- Confirm no lint errors on touched files
