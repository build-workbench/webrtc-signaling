# syntax=docker/dockerfile:1

ARG GO_VERSION=1.23

# ── Build stage ─────────────────────────────────────────
FROM golang:${GO_VERSION}-alpine AS builder
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN apk add --no-cache ca-certificates git && update-ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags "-s -w \
      -X github.com/LessUp/aurora-signal/internal/version.Version=${VERSION} \
      -X github.com/LessUp/aurora-signal/internal/version.Commit=${COMMIT} \
      -X github.com/LessUp/aurora-signal/internal/version.BuildTime=${BUILD_TIME}" \
    -o /out/signal-server ./cmd/server

# ── Runtime stage ───────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot AS runtime
ARG VERSION=dev
LABEL org.opencontainers.image.title="aurora-signal" \
      org.opencontainers.image.description="WebRTC signaling server" \
      org.opencontainers.image.source="https://github.com/LessUp/aurora-signal" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.licenses="MIT"
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /out/signal-server /app/signal-server
COPY web /app/web
ENV SIGNAL_ADDR=:8080
EXPOSE 8080 9090
USER nonroot:nonroot
ENTRYPOINT ["/app/signal-server"]
