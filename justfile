
registry := "registry.digitalocean.com/bidon-io"
tag := `git rev-parse HEAD`

dev:
    docker compose -f docker-compose.dev.yml up

seed:
    go run ./cmd/bidon-seed -reset

sdk-api:
    go run ./cmd/bidon-sdkapi

#

test-db:
    docker compose up migrate-test

test:
    go test ./...

#

[arg('env', pattern='local|test|staging|prod')]
switch env:
    ln -sf .env.{{env}} .env

#

docker-login:
    doctl registry login

_build target:
    just docker-login
    docker buildx build --platform linux/amd64 --target {{target}} --tag {{registry}}/{{target}}:{{tag}} --push .

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
