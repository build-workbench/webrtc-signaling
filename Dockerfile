# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS builder
RUN apk add --no-cache ca-certificates && update-ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /out/signal-server ./cmd/server

# Distroless runtime
FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=builder /out/signal-server /signal-server
COPY web /web
ENV SIGNAL_ADDR=:8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/signal-server"]
