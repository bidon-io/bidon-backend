# Local development

Commands below run from the repository root.

## Nix

Nix keeps the development environment reproducible.

1. Install [Nix](https://nixos.org/download/)
2. Install [direnv](https://direnv.net/docs/installation.html)
3. Run `direnv allow`. The nix flake shell will load.

## Full local stack

Runs Postgres, Redis, Redpanda (Kafka API), migrations, seed data, both API services, the DSP simulator, and the Nuxt frontend in one command:

```shell
docker compose -f docker-compose.dev.yml up -d
```

`just compose` does the same in the foreground, and `just compose-down` stops it.

| Service           | URL                        |
|-------------------|----------------------------|
| bidon-ui          | http://localhost:3010      |
| bidon-admin       | http://localhost:1323      |
| bidon-sdkapi      | http://localhost:1324 (gRPC localhost:50051) |
| bidon-dspsim      | http://localhost:1325      |
| Postgres          | localhost:5434             |
| Redis             | localhost:6379             |
| Redpanda (Kafka)  | localhost:19092            |
| Redpanda Console  | http://localhost:8080      |
| VictoriaMetrics   | http://localhost:8428 (see [telemetry/local-stack.md](../telemetry/local-stack.md)) |

**First run** requires internet access to pull images and download Go modules. Subsequent runs work offline once the module cache is warm.
Frontend (`bidon-ui`) runs with file watching enabled for Docker and hot-reloads on changes under `web/bidon_ui/`.
API requests from the frontend (`/api/**`, `/auth/**`) are proxied to `bidon-admin` automatically in this setup.

To re-run migrations and seed data (for example after a reset):

```shell
docker compose -f docker-compose.dev.yml rm -f migrate bidon-seed && \
docker compose -f docker-compose.dev.yml up -d
```

To tear down and remove all data:

```shell
docker compose -f docker-compose.dev.yml down --volumes --remove-orphans
```

## Manual setup (dependencies only)

To run the services directly on your machine:

```shell
make local-init

docker compose up -d
```

To tear down this stack and remove its data and locally built images:

```shell
docker compose down --volumes --rmi local --remove-orphans || true
```

## Migrations

```shell
go run ./cmd/bidon-migrate -help
```

## Admin API

```shell
go run ./cmd/bidon-admin   # :1323 (ADMIN_PORT)
```

## sdkapi

```shell
go run ./cmd/bidon-sdkapi  # :1324 (SDKAPI_PORT), gRPC :50051 (GRPC_PORT)
```

Each service reads its own port variable first, then the shared `PORT`, then its
default, so admin and sdkapi can run side by side. If your `.env.local` still has
`PORT=1323` from an older `.env.sample`, add `ADMIN_PORT` / `SDKAPI_PORT`
(`just config-diff` lists them) or remove `PORT`.

## Tests

On your machine, start the test database first:

```shell
just test-db
just test
```

Or run the suite inside Docker:

```shell
make test
```

The end-to-end SDK suite has its own stack (`just test-e2e-up`, `just test-e2e`).
See "End-to-end SDK flow" in [AGENTS.md](../../AGENTS.md).

## Read from Kafka

Full local stack (Redpanda), or browse topics in Redpanda Console at http://localhost:8080:

```shell
docker compose -f docker-compose.dev.yml exec redpanda rpk topic consume ad-events
```

Manual setup (`docker-compose.yml`, Kafka on localhost:9092):

```shell
docker compose exec -it kafka kafka-console-consumer --bootstrap-server=localhost:9092 --topic=ad-events --from-beginning
```

## Copilot

See [copilot/README.md](../../copilot/README.md).
