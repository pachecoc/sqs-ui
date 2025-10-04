# Binary / image naming
APP_NAME = sqs-ui
IMAGE = ghcr.io/pachecoc/$(APP_NAME)
# Runtime defaults
PORT ?= 8080
QUEUE_NAME ?= example
# Build flags
LDFLAGS = -s -w
GOFLAGS = -trimpath

.PHONY: build run-local docker-build docker-push fmt tidy test lint

build:
	@echo "Building $(APP_NAME)..."
	CGO_ENABLED=0 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o bin/$(APP_NAME) ./cmd/server

run-local:
	@echo "Running $(APP_NAME) locally..."
	QUEUE_NAME=$(QUEUE_NAME) PORT=$(PORT) go run ./cmd/server

docker-build:
	docker buildx build --platform linux/amd64,linux/arm64 -t $(IMAGE):latest .

docker-push:
	docker push $(IMAGE):latest

fmt:
	go fmt ./...

tidy:
	go mod tidy

lint:
	go vet ./...

test:
	go test -count=1 ./...
