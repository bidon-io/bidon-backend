# Build UI
FROM node:22-alpine AS frontend-deps

WORKDIR /app

COPY web/bidon_ui/package.json web/bidon_ui/yarn.lock ./
RUN yarn install --frozen-lockfile

ARG APP_ENV=production
COPY web/bidon_ui .
RUN VITE_APP_ENV=${APP_ENV} yarn generate

FROM node:22-alpine AS bidon-ui-builder

WORKDIR /app

COPY web/bidon_ui/package.json web/bidon_ui/yarn.lock ./
RUN yarn install --frozen-lockfile

ARG APP_ENV=production
COPY web/bidon_ui .
RUN VITE_APP_ENV=${APP_ENV} yarn build

FROM node:22-alpine AS bidon-ui

WORKDIR /app

COPY --from=bidon-ui-builder /app/.output ./.output

EXPOSE 3000

CMD ["node", ".output/server/index.mjs"]

FROM golang:1.24-alpine AS base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

FROM base AS bidon-admin-builder

COPY --from=frontend-deps /app/.output/public ./cmd/bidon-admin/web/ui
RUN go build -o /bidon-admin ./cmd/bidon-admin

FROM base AS bidon-sdkapi-builder

RUN go build -o /bidon-sdkapi ./cmd/bidon-sdkapi

FROM base AS bidon-migrate-builder

RUN go build -o /bidon-migrate ./cmd/bidon-migrate

FROM base AS bidon-seed-builder

RUN go build -o /bidon-seed ./cmd/bidon-seed

FROM base AS bidon-dspsim-builder

RUN go build -o /bidon-dspsim ./cmd/bidon-dspsim

FROM rust:1.83-alpine AS proxy-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache musl-dev gcc make linux-headers protobuf-dev openssl-dev perl

# Copy Cargo files first to cache dependencies
COPY Cargo.toml Cargo.lock ./
RUN mkdir -p proxy/src \
    && echo "fn main() {}" > proxy/src/main.rs \
    && touch proxy/src/lib.rs
RUN cargo build --release
RUN rm -rf proxy/src

# Copy actual source code and build
COPY proxy/src ./proxy/src
COPY proto ./proto
COPY build.rs ./build.rs
RUN cargo build --release

FROM alpine:3.18 AS deploy

RUN apk add --no-cache ca-certificates

RUN adduser -D -u 1000 deploy
USER deploy

EXPOSE 1323
EXPOSE 50051

FROM deploy AS bidon-admin

COPY --from=bidon-admin-builder --chown=deploy /bidon-admin /bidon-admin

CMD [ "/bidon-admin" ]

FROM deploy AS bidon-sdkapi

COPY --from=bidon-sdkapi-builder --chown=deploy /bidon-sdkapi /bidon-sdkapi

CMD [ "/bidon-sdkapi" ]

FROM deploy AS bidon-migrate

COPY --from=bidon-migrate-builder --chown=deploy /bidon-migrate /bidon-migrate

ENTRYPOINT [ "/bidon-migrate" ]

CMD [ "status" ]

FROM deploy AS bidon-seed

COPY --from=bidon-seed-builder --chown=deploy /bidon-seed /bidon-seed

CMD [ "/bidon-seed" ]

FROM deploy AS bidon-dspsim

COPY --from=bidon-dspsim-builder --chown=deploy /bidon-dspsim /bidon-dspsim

CMD [ "/bidon-dspsim" ]

EXPOSE 1325

FROM deploy AS bidon-proxy

COPY --from=proxy-builder --chown=deploy /app/target/release/bidon-proxy /bidon-proxy

CMD [ "/bidon-proxy" ]

EXPOSE 3000
