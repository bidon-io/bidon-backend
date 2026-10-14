# syntax=docker/dockerfile:1

FROM rust:1.98-bookworm AS builder

WORKDIR /app

COPY telemetry/Cargo.toml telemetry/Cargo.lock ./
COPY schemas/proto /schemas/proto
RUN mkdir src \
    && echo "fn main() {}" > src/main.rs \
    && echo "" > src/lib.rs \
    && cargo build --release \
    && rm -rf src target/release/bidon-telemetry target/release/deps/bidon_telemetry*

COPY telemetry /app
COPY schemas/proto /schemas/proto
RUN cargo build --release

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --uid 1000 --create-home deploy

COPY --from=builder --chown=deploy /app/target/release/bidon-telemetry /bidon-telemetry

USER deploy

EXPOSE 1326

CMD ["/bidon-telemetry"]
