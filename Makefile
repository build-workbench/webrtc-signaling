GO ?= go
BIN ?= bin/signal-server
PKG := ./...
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -s -w \
	-X signal/internal/version.Version=$(VERSION) \
	-X signal/internal/version.Commit=$(COMMIT) \
	-X signal/internal/version.BuildTime=$(BUILD_TIME)

.PHONY: all deps build run test test-race test-cover vet lint docker-build compose-up compose-down

all: build

deps:
	$(GO) mod tidy

build:
	mkdir -p bin
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/server

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

docker-build:
	docker build -t lessup/signaling:dev .

compose-up:
	cd docker && docker compose up --build -d

compose-down:
	cd docker && docker compose down
