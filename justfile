
registry := "registry.digitalocean.com/bidon-io"
tag := `git rev-parse --short HEAD`

# UI deploy — set in .env or export before running
ui_bucket       := env('SPACES_STAGING_BUCKET', '')
ui_region       := env('SPACES_REGION', 'ams3')
do_access_key   := env('DO_ACCESS_KEY_ID', '')
do_secret_key   := env('DO_SECRET_ACCESS_KEY', '')

dev:
    docker compose -f docker-compose.dev.yml up

seed:
    go run ./cmd/bidon-seed -reset

sdk-api:
    go run ./cmd/bidon-sdkapi

#

[arg('env', pattern='local|staging|prod')]
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

build-proxy:
    just _build bidon-proxy

build-all:
    just build-admin
    just build-sdkapi
    just build-migrate
    just build-seed
    just build-proxy

# Build the Nuxt SPA and sync to the staging Spaces bucket.
# Required env vars: SPACES_STAGING_BUCKET, DO_ACCESS_KEY_ID, DO_SECRET_ACCESS_KEY
# Optional: SPACES_REGION (default: ams3)
deploy-staging-ui:
    #!/usr/bin/env sh
    set -e
    [ -z "{{ui_bucket}}" ]     && echo "SPACES_STAGING_BUCKET is required" && exit 1
    [ -z "{{do_access_key}}" ] && echo "DO_ACCESS_KEY_ID is required"      && exit 1
    [ -z "{{do_secret_key}}" ] && echo "DO_SECRET_ACCESS_KEY is required"  && exit 1
    cd web/bidon_ui && yarn generate
    AWS_ACCESS_KEY_ID="{{do_access_key}}" AWS_SECRET_ACCESS_KEY="{{do_secret_key}}" \
        aws s3 sync .output/public/ \
        "s3://{{ui_bucket}}/{{tag}}/" \
        --endpoint-url "https://{{ui_region}}.digitaloceanspaces.com" \
        --acl public-read \
        --delete
    echo "Deployed → s3://{{ui_bucket}}/{{tag}}/"
