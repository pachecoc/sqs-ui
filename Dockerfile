# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/sqs-ui ./cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=builder /out/sqs-ui /sqs-ui
COPY web /web
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/sqs-ui"]
