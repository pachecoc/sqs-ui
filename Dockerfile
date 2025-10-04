# syntax=docker/dockerfile:1.6

### --- Build Stage ---
FROM golang:1.24-alpine AS builder
WORKDIR /src

# Optimize module caching
COPY go.mod go.sum ./
RUN go mod download

# Copy app source
COPY . .

# Build statically
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /out/sqs-ui ./cmd/server

### --- Final Stage ---
FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=builder /out/sqs-ui /sqs-ui
COPY internal/templates /internal/templates

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/sqs-ui"]
