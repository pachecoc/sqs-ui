FROM golang:1.23 AS builder
WORKDIR /app
COPY . .
RUN go mod download && CGO_ENABLED=0 GOOS=linux go build -o sqs-demo ./cmd/server

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /app/sqs-demo .
COPY internal/templates ./internal/templates
EXPOSE 8080
ENTRYPOINT ["/app/sqs-demo"]
