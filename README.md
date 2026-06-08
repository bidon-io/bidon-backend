# Bidon

## Self-Hosted Bidon Setup

For detailed instructions on setting up a self-hosted instance of Bidon, visit our documentation:

[Self-Hosted Deployment Guide](https://docs.bidon.org/docs/server/self-hosted)

## Copilot (LangGraph) — Local Setup
See the minimal setup guide at `copilot/README.md`.

## Set up a development environment

We use Nix to ensure reproducible development environments.

1. Install [Nix](https://nixos.org/download/#nix-install-macos)
2. Install [direnv](https://direnv.net/docs/installation.html)
3. Run `direnv allow` the nix flake shell will load

### Full local stack (recommended)

Runs Postgres, Redis, Kafka, migrations, seed data, both API services, and the Nuxt frontend in one command:

```shell
docker compose -f docker-compose.dev.yml up -d
```

| Service     | URL                        |
|-------------|----------------------------|
| bidon-ui    | http://localhost:3010      |
| bidon-admin | http://localhost:1323      |
| bidon-sdkapi| http://localhost:1324      |
| Postgres    | localhost:5434             |
| Kafka       | localhost:9092             |

**First run** requires internet access to pull images and download Go modules. Subsequent runs work offline once the module cache is warm.
Frontend (`bidon-ui`) runs with file watching enabled for Docker and hot-reloads on changes under `web/bidon_ui/`.
API requests from the frontend (`/api/**`, `/auth/**`) are proxied to `bidon-admin` automatically in this setup.

To re-run migrations and seed data (e.g. after a reset):
```shell
docker compose -f docker-compose.dev.yml rm -f migrate bidon-seed && \
docker compose -f docker-compose.dev.yml up -d
```

To tear down and remove all data:
```shell
docker compose -f docker-compose.dev.yml down --volumes --remove-orphans
```

### Manual setup (dependencies only)

If you prefer to run the services directly on your machine:

```shell
make local-init

docker compose up -d
```

### Manage migrations
```shell
go run ./cmd/bidon-migrate -help
```

### Start admin backend
```shell
go run ./cmd/bidon-admin
```

### Start sdkapi backend
```shell
go run ./cmd/bidon-sdkapi
```

### Run tests
```shell
make test
```

## Docker images

Images are tagged with the current git SHA and named `registry.digitalocean.com/bidon-io/<service>:<sha>`.

### Local builds (testing)

Build images into your local Docker store for `docker run` or compose testing. No registry login or push:

```shell
just build-admin    # bidon-admin
just build-sdkapi   # bidon-sdkapi
just build-migrate  # bidon-migrate
just build-seed     # bidon-seed
just build-all      # all of the above
```

### CI / registry builds (staging and deploy)

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
```

Or push all services (logs in once):

```shell
just ci-build-all
```

### Staging deployment

Staging runs in [Coolify](http://coolify.bidon.squads.com/).

After pushing images with `just ci-build-*`, update the app configuration there — set each `BIDON_*_TAG` environment variable to the git SHA produced by the build. See `.env.staging.example` for the full list of required variables.

For initial Coolify setup (Terraform provisioning, `bidon-coolify` CLI), see `infra/README.md`.

### Clean env
```shell
docker compose down --volumes --rmi local --remove-orphans || true
```

### Read from kafka
```shell
docker compose exec -it kafka kafka-console-consumer --bootstrap-server=localhost:9092 --topic=bidon-ad-events --from-beginning
```
