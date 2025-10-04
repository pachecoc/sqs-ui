APP_NAME = sqs-ui
IMAGE = ghcr.io/pachecoc/$(APP_NAME)
PORT ?= 8080
QUEUE_NAME ?= example

build:
	@echo "🚧 Building $(APP_NAME)..."
	go build -o bin/$(APP_NAME) ./cmd/server

run-local:
	@echo "🏃 Running $(APP_NAME) locally..."
	QUEUE_NAME=$(QUEUE_NAME) go run ./cmd/server

docker-build:
	docker buildx build --platform linux/amd64,linux/arm64 -t $(IMAGE):latest .

docker-push:
	docker push $(IMAGE):latest

fmt:
	go fmt ./...

tidy:
	go mod tidy
