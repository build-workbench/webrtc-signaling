GO ?= go
BIN ?= bin/signal-server
PKG := ./...

.PHONY: all deps build run test lint docker-build compose-up compose-down

all: build

deps:
	$(GO) mod tidy

build:
	mkdir -p bin
	$(GO) build -o $(BIN) ./cmd/server

run:
	SIGNAL_JWT_SECRET=$${SIGNAL_JWT_SECRET:-dev-secret-change} $(GO) run ./cmd/server

test:
	$(GO) test $(PKG)

lint:
	golangci-lint run || echo "Install golangci-lint for linting"

docker-build:
	docker build -t lessup/signaling:dev .

compose-up:
	cd docker && docker compose up --build -d

compose-down:
	cd docker && docker compose down
