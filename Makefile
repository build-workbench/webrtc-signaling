GO ?= go
BIN ?= bin/signal-server
PKG := ./...
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -s -w \
	-X main.Version=$(VERSION) \
	-X main.Commit=$(COMMIT) \
	-X main.BuildTime=$(BUILD_TIME)

.PHONY: build run test test-race test-cover vet fmt clean

build:
	mkdir -p bin
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/server

run:
	$(GO) run ./cmd/server

test:
	$(GO) test $(PKG)

test-race:
	$(GO) test -race $(PKG)

test-cover:
	$(GO) test -coverprofile=coverage.out $(PKG)
	$(GO) tool cover -html=coverage.out -o coverage.html

vet:
	$(GO) vet $(PKG)

fmt:
	$(GO) fmt $(PKG)

clean:
	rm -rf bin/ coverage.out coverage.html
