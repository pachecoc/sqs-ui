# syntax=docker/dockerfile:1

# -------------------------------
# 🏗️ Stage 1: Build static Go binary
# -------------------------------
FROM golang:1.23-alpine AS builder

WORKDIR /src

# Copy dependency files and download modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Set build arguments for metadata (provided via Makefile)
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_TIME=unknown

# Inject metadata into the binary
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w \
    -X github.com/pachecoc/sqs-ui/internal/version.Version=${VERSION} \
    -X github.com/pachecoc/sqs-ui/internal/version.Commit=${COMMIT} \
    -X github.com/pachecoc/sqs-ui/internal/version.BuildTime=${BUILD_TIME}" \
    -o /out/sqs-ui ./cmd/server

# -------------------------------
# 🚀 Stage 2: Minimal runtime image
# -------------------------------
FROM gcr.io/distroless/static-debian12

# Copy compiled binary and web assets
COPY --from=builder /out/sqs-ui /sqs-ui
COPY web /web

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/sqs-ui"]
