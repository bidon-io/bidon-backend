
registry := "registry.digitalocean.com/bidon-io"
tag := `git rev-parse HEAD`

# --- Development ---

compose:
    docker compose -f docker-compose.dev.yml up

compose-down:
    docker compose -f docker-compose.dev.yml down --remove-orphans

# Apply migrations. Pass args to override, e.g. `just migrate down`.
migrate *args="up":
    go run ./cmd/bidon-migrate {{args}}

seed:
    go run ./cmd/bidon-seed -reset -sample

admin:
    go run ./cmd/bidon-admin

sdk-api:
    go run ./cmd/bidon-sdkapi

dsp-sim:
    go run ./cmd/bidon-dspsim

# --- Git workflow ---

# One-time repo setup for stacked PRs (see docs/dev/git-spice.md)
spice-init:
    gs repo init --trunk new-main --remote origin
    git config --local spice.submit.navigationComment multiple
    git config --local spice.repoSync.restack upstack
    git config --local spice.merge.method squash
    git config --local spice.log.pushStatusFormat aheadBehind
    git config --local spice.branchPrompt.sort -committerdate
    git config --local spice.submit.web created

# --- Testing ---

test-db:
    docker compose up migrate-test

test:
    go test ./...

# Bring up Redis + the DSP simulator (and its Postgres + migrations) for the e2e suite.
test-e2e-up:
    docker compose -f docker-compose.test.yml up -d redis dspsim

test-e2e-down:
    docker compose -f docker-compose.test.yml down --remove-orphans

# Run the e2e suite against a stack started with `just test-e2e-up`.
test-e2e:
    go test -tags e2e ./internal/sdkapi/v2/e2e/...

# Run the e2e suite fully inside Docker, the way CI does.
test-e2e-ci:
    docker compose -f docker-compose.test.yml run --rm go-e2e

precommit:
    pre-commit run --all-files

# --- Config ---

# Ensures .env.local exists
config-exists:
    #!/usr/bin/env sh
    if [ ! -f .env.local ]; then
      cp .env.sample .env.local && echo "Copied .env.sample → .env.local"
    fi

# Prints keys defined in .env.sample that are missing from .env.local
config-diff: config-exists
    #!/usr/bin/env bash
    EXAMPLE=".env.sample"
    LOCAL=".env.local"

    extract_keys() {
        grep -E '^[A-Z0-9_]+=' "$1" | cut -d= -f1
    }

    missing=()
    while IFS= read -r key; do
        if ! grep -qE "^#?${key}=" "$LOCAL" 2>/dev/null; then
            missing+=("$key")
        fi
    done < <(extract_keys "$EXAMPLE")

    if [[ ${#missing[@]} -gt 0 ]]; then
        echo -e "\nWARNING: Keys in $EXAMPLE missing from $LOCAL:"
        printf '  %s\n' "${missing[@]}"
        echo -e "\n"
    fi


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

build-dspsim:
    just _build bidon-dspsim

build-ui:
    just _build bidon-ui

build-all:
    just build-admin
    just build-sdkapi
    just build-migrate
    just build-seed
    just build-dspsim
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

ci-build-dspsim:
    just _ci-build bidon-dspsim

ci-build-ui:
    just _ci-build bidon-ui

ci-build-all:
    just docker-login
    just ci-build-admin
    just ci-build-sdkapi
    just ci-build-migrate
    just ci-build-seed
    just ci-build-dspsim
    just ci-build-ui
