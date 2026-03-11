# Stage 1: Build
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -ldflags "-s -w \
    -X main.version=$(git describe --tags --always 2>/dev/null || echo dev) \
    -X main.commit=$(git rev-parse --short HEAD 2>/dev/null || echo none) \
    -X main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /kantar ./cmd/kantar

RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /kantarctl ./cmd/kantarctl

# Stage 2: Runtime
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 1000 kantar

COPY --from=builder /kantar /usr/local/bin/kantar
COPY --from=builder /kantarctl /usr/local/bin/kantarctl

RUN mkdir -p /var/lib/kantar/data /var/lib/kantar/db /var/lib/kantar/logs /etc/kantar && \
    chown -R kantar:kantar /var/lib/kantar /etc/kantar

USER kantar

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/healthz || exit 1

ENTRYPOINT ["kantar"]
CMD ["serve", "--config", "/etc/kantar/kantar.toml"]
