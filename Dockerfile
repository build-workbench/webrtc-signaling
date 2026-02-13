# syntax=docker/dockerfile:1

ARG GO_VERSION=1.23

FROM golang:${GO_VERSION}-alpine AS builder
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown
RUN apk add --no-cache ca-certificates git && update-ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags "-s -w \
      -X signal/internal/version.Version=${VERSION} \
      -X signal/internal/version.Commit=${COMMIT} \
      -X signal/internal/version.BuildTime=${BUILD_TIME}" \
    -o /out/signal-server ./cmd/server

# Distroless runtime
FROM gcr.io/distroless/base-debian12
LABEL org.opencontainers.image.title="aurora-signal" \
      org.opencontainers.image.description="WebRTC signaling server" \
      org.opencontainers.image.source="https://github.com/LessUp/aurora-signal"
WORKDIR /
COPY --from=builder /out/signal-server /signal-server
COPY web /web
ENV SIGNAL_ADDR=:8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/signal-server"]
