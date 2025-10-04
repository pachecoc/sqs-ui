# syntax=docker/dockerfile:1.6

### --- Build Stage ---
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
WORKDIR /src

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Copy all source
COPY . .

# Build static binary for target arch
RUN echo "🔨 Building for $TARGETOS/$TARGETARCH" && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-s -w" -o /out/sqs-ui ./cmd/server

### --- Final Stage ---
FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=builder /out/sqs-ui /sqs-ui
COPY internal/templates /internal/templates
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/sqs-ui"]
