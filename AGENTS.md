# Agent Instructions

Guidance for AI agents working in the Bidon backend codebase.

## Project Overview

Bidon is a Go-based ad mediation and programmatic advertising platform: RTB
auctions, ad unit management, demand source adapters, win/loss notifications,
and event tracking.

Go · PostgreSQL (GORM) · Redis · Redpanda (Kafka API) · Echo REST + gRPC · Protobuf

## Structure

```
cmd/       bidon-admin, bidon-sdkapi, bidon-migrate, bidon-seed, bidon-coolify
internal/  ad, adapter, admin, auction, audit, bidding, db, device,
           notification, sdkapi, segment
pkg/       reusable packages
proto/     protobuf definitions (git submodule, update=none)
config/    YAML config    docker/  scripts/  web/bidon_ui (Nuxt frontend)
```

## Local Development

Commands are `just` recipes — see `justfile`.

```bash
just compose        # full dev stack: Postgres, Redis, Redpanda,
                    # migrations, seed, both APIs, Nuxt UI (foreground)
just compose-down   # tear down
```

| Service          | URL                   |
|------------------|-----------------------|
| bidon-ui         | http://localhost:3010 |
| bidon-admin      | http://localhost:1323 |
| bidon-sdkapi     | http://localhost:1324 |
| bidon-dspsim     | http://localhost:1325 |
| Postgres         | localhost:5434        |
| Redis            | localhost:6379        |
| Redpanda         | localhost:19092       |
| Redpanda Console | http://localhost:8080 |

```bash
just admin          # admin API only
just sdk-api        # SDK API only
just dsp-sim        # OpenRTB DSP simulator (localhost:1325)
just seed           # reset + load sample data
just migrate        # apply migrations (also: just migrate down)
just config-diff    # ensure .env.local exists, list keys missing vs .env.sample
```

`bidon-dspsim` is a standalone OpenRTB DSP simulator: reads auction config from
Postgres, answers bid requests from a JSON creative library, and records the
nurl/burl/lurl it advertises under `/debug/bids`. Drive it with `dspsim.http`;
pointing a live auction at it needs an out-of-band endpoint redirect. Design:
[docs/adr/0001-dsp-simulator.md](docs/adr/0001-dsp-simulator.md).

First-time setup also needs `make local-init` (submodules + deps).

### Stacked PRs

Trunk is `new-main`, not GitHub's default branch. One-time setup: `gh auth
login` (if needed) then `just spice-init` — `.envrc` hands `gs` gh's token, so
no separate `gs auth login` is required, but run `gs`/`gh` through a
direnv-loaded shell (e.g. `direnv exec .`), not a bare `nix develop`. Daily
loop: `gs branch create` (`bc`), `gs branch submit` (`bs`), `gs repo restack`
(`rr`), `gs repo sync` (`rs`) after a merge — never rebase a stack by hand,
`gs repo sync` reconciles merged branches and restacks what's above them.
Bare `gh pr create` targets `main`; use `gs branch submit` instead. Full
guide: [docs/dev/git-spice.md](docs/dev/git-spice.md).

### Tests

```bash
just test-db        # bring up the test database first
just test           # go test ./...
just precommit      # lint
```

Single package: `go test ./internal/auction/...`
Regenerate mocks: `go generate ./...`

### Images / Coolify

```bash
just build-all      # local docker store
just ci-build-all   # build + push to registry
```

Local dev mounts source into `bidon-ui` and hot-reloads. **Coolify runs pre-built
registry images** tagged via `BIDON_*_TAG` — not the working tree. When debugging
staging-only UI issues, verify tags in Coolify match the commit you expect
(especially `BIDON_UI_TAG`); backend/seed can be newer than `bidon-ui`. Rebuild
with `just ci-build-ui` and redeploy with aligned tags. See README "Staging
deployment".

## Architecture

- **Repository pattern** — `internal/*/store/*_repo.go`, wraps GORM; generic
  `resourceRepo` provides List/Find/Create/Update/Delete
- **Service layer** — `internal/*/service.go`, depends on interfaces, DI
- **Three-layer models** — `internal/db/*.gen.go` (generated) → `internal/admin/*.go`
  (domain) → `internal/admin/openapi/*.go` (API); mappers live in repo files
- **Adapters** — `internal/bidding/adapters/`, common interface per demand source
- **User scoping** — `ListOwnedByUser()` / `FindOwnedByUser()` for multi-tenancy
- **Advisory locking** — `pg_advisory_xact_lock` guards concurrent dedup
  (e.g. `LineItemRepo.firstOrCreate()`)
- **Event logging** — auction/impression/click events to Redpanda

### Auction flow

SDK → `/v2/auction` → segment match + config fetch → build demand (line items +
bidding) → parallel bidding round → ranked ad units → win/loss notifications

## Code Style

- Standard Go conventions; `golangci-lint`
- Interfaces defined in consuming packages
- Context first parameter; explicit errors, no panics in production
- Mocks via `moq` (`go generate ./...`)

## Testing Conventions

`*_test.go`, `testify`, table-driven preferred; DB tests use `internal/db/dbtest`

## Configuration

`.env.sample` → `.env.local` · `config/` · auction configs in the
`auction_configurations` table

## Verification Checklist

- Run targeted tests in changed packages
- Run `just test` when contracts or shared logic change
- Confirm no lint errors on touched files

## Resources

[Self-Hosted Deployment Guide](https://docs.bidon.org/docs/server/self-hosted) ·
OpenAPI: `internal/admin/openapi/` · Proto: `proto/`
