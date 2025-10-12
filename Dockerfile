# syntax=docker/dockerfile:1

# -------------------------------
# Stage 1: Build static Go binary
# -------------------------------
FROM golang:1.23-alpine AS builder

WORKDIR /src

# Copy dependency files and download modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build metadata arguments (can be set at build time):
#   docker build \
#     --build-arg VERSION=$(git describe --tags --always --dirty) \
#     --build-arg COMMIT=$(git rev-parse --short HEAD) \
#     --build-arg BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
#     -t sqs-ui:dev .
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_TIME=unknown

# Produce a static Linux binary (CGO disabled) with version info embedded.
# -trimpath removes local file system paths, -s -w strip debug info.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w \
      -X github.com/pachecoc/sqs-ui/internal/version.Version=${VERSION} \
      -X github.com/pachecoc/sqs-ui/internal/version.Commit=${COMMIT} \
      -X github.com/pachecoc/sqs-ui/internal/version.BuildTime=${BUILD_TIME}" \
    -o /out/sqs-ui ./cmd/server

# (Optional) You can uncomment to see binary size:
# RUN ls -lh /out/sqs-ui

# --------------------------------------------------
# Stage 2: Minimal runtime image
# --------------------------------------------------
FROM gcr.io/distroless/static-debian12

# Re-declare ARGs if you want them available for labels in the final image
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_TIME=unknown

# OCI recommended labels for provenance & traceability
LABEL org.opencontainers.image.title="sqs-ui" \
      org.opencontainers.image.description="Lightweight web UI for interacting with AWS SQS" \
      org.opencontainers.image.source="https://github.com/pachecoc/sqs-ui" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${BUILD_TIME}" \
      org.opencontainers.image.licenses="MIT"

# Explicit working directory (not strictly required for a static binary, but clearer)
WORKDIR /

# Copy compiled binary and static web assets
COPY --from=builder /out/sqs-ui /sqs-ui
COPY web /web

# Expose HTTP port
EXPOSE 8080

# Run as non-root (distroless provides 'nonroot')
USER nonroot:nonroot

# Entrypoint runs the server
ENTRYPOINT ["/sqs-ui"]
