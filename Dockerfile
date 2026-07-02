# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.26 AS builder

# Yandex Mystem is fetched at build time instead of being committed to git.
ARG MYSTEM_URL=https://download.cdn.yandex.net/mystem/mystem-3.1-linux-64bit.tar.gz

WORKDIR /src

# Download and unpack the mystem binary (amd64/glibc)
RUN mkdir -p /out-bin \
    && curl -sSL "$MYSTEM_URL" -o /tmp/mystem.tar.gz \
    && tar -xzf /tmp/mystem.tar.gz -C /out-bin \
    && chmod +x /out-bin/mystem

# Cache dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Build the binary (mystem is a native amd64 dependency, so we target amd64)
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /out/zaglyt-tg .

# ---- Runtime stage ----
# Debian (glibc) is required: the bundled `mystem` binary is dynamically
# linked against glibc and will not run on Alpine/musl.
FROM debian:bookworm-slim

WORKDIR /app

# ca-certificates for outbound TLS; wget for the container healthcheck.
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates wget \
    && rm -rf /var/lib/apt/lists/*

# Application binary
COPY --from=builder /out/zaglyt-tg /app/zaglyt-tg

# Yandex Mystem binary fetched in the build stage (looked up as ./mystem at runtime)
COPY --from=builder /out-bin/mystem /app/mystem

# Directory where per-channel message files are stored (mount as a volume)
RUN mkdir -p /app/database/messages

# Webhook HTTP server port (only used when BOT_MODE=webhook)
EXPOSE 8080

ENTRYPOINT ["/app/zaglyt-tg"]
