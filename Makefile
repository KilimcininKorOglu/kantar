# Kantar - Unified Local Package Registry Platform
# Build system

BINARY_NAME := kantar
CLI_NAME := kantarctl
MODULE := github.com/KilimcininKorOglu/kantar

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)"

# Go settings
GO := go
GOFLAGS :=
GOTEST := $(GO) test
GOVET := $(GO) vet
GOFMT := gofumpt
GOLINT := golangci-lint

# Output
BUILD_DIR := bin
COVERAGE_DIR := coverage

.PHONY: all build build-cli clean test test-cover lint fmt vet run help generate web docker-up docker-down docker-rebuild docker-logs

## Default target
all: lint test build

## Build the kantar server binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/kantar

## Build the kantarctl CLI binary
build-cli:
	@echo "Building $(CLI_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(CLI_NAME) ./cmd/kantarctl

## Build both binaries
build-all: build build-cli

## Remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) $(COVERAGE_DIR)
	$(GO) clean -cache -testcache

## Run all tests
test:
	$(GOTEST) ./... -race -count=1

## Run tests with coverage
test-cover:
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) ./... -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic
	$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report: $(COVERAGE_DIR)/coverage.html"

## Run linter
lint:
	$(GOLINT) run ./...

## Format code
fmt:
	$(GO) fmt ./...
	@command -v $(GOFMT) >/dev/null 2>&1 && $(GOFMT) -w . || true

## Run go vet
vet:
	$(GOVET) ./...

## Generate code (sqlc, etc.)
generate:
	$(GO) generate ./...

## Run the server locally
run: build
	./$(BUILD_DIR)/$(BINARY_NAME) serve

## Run with go run
dev:
	$(GO) run $(LDFLAGS) ./cmd/kantar serve

## Tidy dependencies
tidy:
	$(GO) mod tidy

## Cross-compilation targets
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: build-cross
## Build for all platforms
build-cross:
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} ; \
		output=$(BUILD_DIR)/$(BINARY_NAME)-$${GOOS}-$${GOARCH} ; \
		if [ "$${GOOS}" = "windows" ]; then output=$${output}.exe; fi ; \
		echo "Building $${output}..." ; \
		GOOS=$${GOOS} GOARCH=$${GOARCH} CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o $${output} ./cmd/kantar || exit 1 ; \
	done

## Build web UI
web:
	cd web && npm run build

## Docker: build and start
docker-up:
	docker compose down
	docker compose up --build -d

## Docker: stop
docker-down:
	docker compose down

## Docker: full rebuild without cache
docker-rebuild:
	docker compose down
	docker compose build --no-cache
	docker compose up -d

## Docker: show logs
docker-logs:
	docker compose logs kantar

## Show help
help:
	@echo "Kantar Build System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
	@echo ""
	@echo "  build        Build kantar server binary"
	@echo "  build-cli    Build kantarctl CLI binary"
	@echo "  build-all    Build both binaries"
	@echo "  test         Run tests"
	@echo "  test-cover   Run tests with coverage report"
	@echo "  lint         Run golangci-lint"
	@echo "  fmt          Format code"
	@echo "  vet          Run go vet"
	@echo "  generate     Run go generate"
	@echo "  run          Build and run server"
	@echo "  dev          Run server via go run"
	@echo "  tidy         Tidy Go modules"
	@echo "  build-cross  Cross-compile for all platforms"
	@echo "  web            Build web UI"
	@echo "  docker-up      Build and start Docker"
	@echo "  docker-down    Stop Docker"
	@echo "  docker-rebuild Full rebuild without cache"
	@echo "  docker-logs    Show kantar logs"
	@echo "  clean          Remove build artifacts"
	@echo "  help           Show this help"
