# syntax=docker/dockerfile:1

# Stage 1: Build static Go binary
FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/sqs-ui ./cmd/server

# Stage 2: Minimal runtime image
FROM gcr.io/distroless/static-debian12
COPY --from=builder /out/sqs-ui /sqs-ui
COPY web /web
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/sqs-ui"]
