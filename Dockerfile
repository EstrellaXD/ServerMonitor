FROM node:20-alpine AS frontend

WORKDIR /build
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

FROM rust:1.83-slim AS backend

WORKDIR /build

RUN apt-get update && apt-get install -y --no-install-recommends \
    pkg-config \
    libssl-dev \
    && rm -rf /var/lib/apt/lists/*

COPY backend/Cargo.toml backend/Cargo.lock ./
RUN mkdir src && echo "fn main() {}" > src/main.rs && echo "" > src/lib.rs
RUN cargo build --release 2>/dev/null || true
RUN rm -rf src

COPY backend/src/ src/
RUN touch src/main.rs src/lib.rs
RUN cargo build --release

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libssl3 \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=backend /build/target/release/server-monitor /app/server-monitor
COPY --from=frontend /build/dist /app/static

EXPOSE 8742

CMD ["/app/server-monitor"]
