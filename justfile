
registry := "registry.digitalocean.com/bidon-io"
tag := `git rev-parse HEAD`

# --- Development ---

dev:
    docker compose -f docker-compose.dev.yml up

dev-down:
    docker compose -f docker-compose.dev.yml down --remove-orphans

seed:
    go run ./cmd/bidon-seed -reset -sample

sdk-api:
    go run ./cmd/bidon-sdkapi

dsp-simulator:
    go run ./cmd/bidon-dspsimulator

#

test-db:
    docker compose up migrate-test

test:
    go test ./...

precommit:
    pre-commit run --all-files

#

[arg('env', pattern='local|test|staging|prod')]
switch env:
    ln -sf .env.{{env}} .env

# --- Local image builds ---
# Build images into the local Docker store (--load) for testing with docker run / compose.
# No registry login or push. Images are tagged with the same name as the registry for parity.

_build target:
    docker buildx build --load --platform linux/amd64 --target {{target}} --tag {{registry}}/{{target}}:{{tag}} .

build-admin:
    just _build bidon-admin

build-sdkapi:
    just _build bidon-sdkapi

build-migrate:
    just _build bidon-migrate

build-seed:
    just _build bidon-seed

build-ui:
    just _build bidon-ui

build-all:
    just build-admin
    just build-sdkapi
    just build-migrate
    just build-seed
    just build-ui

# --- CI / registry image builds ---
# Build and push images to the DigitalOcean Container Registry (--push).
# Run `just docker-login` before individual ci-build-* recipes; ci-build-all logs in once.

docker-login:
    doctl registry login

_ci-build target:
    docker buildx build --platform linux/amd64 --target {{target}} --tag {{registry}}/{{target}}:{{tag}} --push .

ci-build-admin:
    just _ci-build bidon-admin

ci-build-sdkapi:
    just _ci-build bidon-sdkapi

ci-build-migrate:
    just _ci-build bidon-migrate

ci-build-seed:
    just _ci-build bidon-seed

ci-build-ui:
    just _ci-build bidon-ui

ci-build-all:
    just docker-login
    just ci-build-admin
    just ci-build-sdkapi
    just ci-build-migrate
    just ci-build-seed
    just ci-build-ui
