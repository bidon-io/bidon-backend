
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
