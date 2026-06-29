# syntax=docker/dockerfile:1

# --- Build stage ---
# GOTOOLCHAIN=auto: 自动拉取 go.mod 要求的 Go 版本（兼容镜像源无 1.26 标签）
FROM golang:1.23-alpine AS builder

ENV GOTOOLCHAIN=auto
RUN apk add --no-cache git ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /surge-host \
    ./cmd/server

# --- Runtime stage ---
FROM alpine:3.21

RUN apk add --no-cache git ca-certificates tzdata wget \
    && adduser -D -u 1000 -h /app surge

WORKDIR /app

COPY --from=builder /surge-host /app/surge-host
COPY web/ /app/web/

RUN mkdir -p /app/data \
    && chown -R surge:surge /app

USER surge

EXPOSE 8080

ENV SURGE_HOST_PORT=8080 \
    SURGE_HOST_DATA_DIR=/app/data \
    SURGE_HOST_STATIC_DIR=/app/web/static \
    SURGE_HOST_TEMPLATES_DIR=/app/web/templates

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1

ENTRYPOINT ["/app/surge-host"]