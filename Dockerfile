# Stage 1: Web UI Build
FROM node:22-alpine AS web-builder

WORKDIR /web

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ .
RUN npm run build

# Stage 2: Go Build
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=web-builder /web/dist ./web/dist

RUN CGO_ENABLED=1 go build -ldflags "-s -w \
    -X main.version=$(git describe --tags --always 2>/dev/null || echo dev) \
    -X main.commit=$(git rev-parse --short HEAD 2>/dev/null || echo none) \
    -X main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /kantar ./cmd/kantar

RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /kantarctl ./cmd/kantarctl

# Stage 3: Runtime
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 1000 kantar

COPY --from=builder /kantar /usr/local/bin/kantar
COPY --from=builder /kantarctl /usr/local/bin/kantarctl

RUN mkdir -p /var/lib/kantar/data /var/lib/kantar/logs /etc/kantar && \
    chown -R kantar:kantar /var/lib/kantar /etc/kantar

USER kantar

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1

ENTRYPOINT ["kantar"]
CMD ["serve", "--config", "/etc/kantar/kantar.toml"]
