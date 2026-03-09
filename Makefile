GO ?= go
BIN ?= bin/signal-server
PKG := ./...
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -s -w \
	-X github.com/LessUp/aurora-signal/internal/version.Version=$(VERSION) \
	-X github.com/LessUp/aurora-signal/internal/version.Commit=$(COMMIT) \
	-X github.com/LessUp/aurora-signal/internal/version.BuildTime=$(BUILD_TIME)

.PHONY: all deps build run test test-race test-cover vet lint fmt clean docker-build docker-push compose-up compose-down compose-logs

all: build

deps:
	$(GO) mod tidy

build:
	mkdir -p bin
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/server

run:
	SIGNAL_JWT_SECRET=$${SIGNAL_JWT_SECRET:-dev-secret-change} $(GO) run ./cmd/server

test:
	$(GO) test $(PKG)

test-race:
	$(GO) test -race $(PKG)

test-cover:
	$(GO) test -coverprofile=coverage.out $(PKG)
	$(GO) tool cover -html=coverage.out -o coverage.html

vet:
	$(GO) vet $(PKG)

lint:
	golangci-lint run || echo "Install golangci-lint for linting"

fmt:
	$(GO) fmt $(PKG)

clean:
	rm -rf bin/ coverage.out coverage.html

docker-build:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t lessup/signaling:$(VERSION) \
		-t lessup/signaling:latest .

docker-push:
	docker push lessup/signaling:$(VERSION)
	docker push lessup/signaling:latest

compose-up:
	cd docker && docker compose up --build -d

compose-down:
	cd docker && docker compose down -v

compose-logs:
	cd docker && docker compose logs -f
