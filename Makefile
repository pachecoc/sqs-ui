# --------------------------------------
# 🧱 SQS-UI  |  Docker Hub Edition
# --------------------------------------

DOCKER_USER ?= pachecoc
IMAGE_NAME ?= $(DOCKER_USER)/sqs-ui
TAG ?= latest
PLATFORMS ?= linux/amd64,linux/arm64
BUILDER ?= multiarch-builder
IMAGE_TAGGED := $(IMAGE_NAME):$(TAG)

# --- Default ---
all: tidy build

# --------------------------------------
# ⚙️ Go Development
# --------------------------------------
init:
	@echo "🚀 Reinitializing Go module..."
	rm -f go.mod go.sum
	go mod init github.com/$(DOCKER_USER)/sqs-ui
	go mod tidy

tidy:
	@echo "📦 Tidying Go modules..."
	go mod tidy

verify:
	go mod verify

build-local:
	CGO_ENABLED=0 go build -o bin/sqs-ui ./cmd/server

run-local:
	QUEUE_NAME=my-demo-queue go run ./cmd/server

clean-go:
	rm -rf bin/ go.sum

# --------------------------------------
# 🐳 Docker Build & Push
# --------------------------------------
build:
	@echo "🚧 Building $(IMAGE_TAGGED)..."
	docker build -t $(IMAGE_TAGGED) .

run:
	docker run --rm -it -p 8080:8080 -e QUEUE_NAME=my-demo-queue $(IMAGE_TAGGED)

builder:
	@echo "🧱 Creating buildx builder..."
	-docker buildx create --name $(BUILDER) --use
	docker buildx inspect --bootstrap

buildx: builder
	@echo "🚀 Building multi-arch image..."
	docker buildx build --platform $(PLATFORMS) -t $(IMAGE_TAGGED) --push .

login:
	@echo "🔐 Logging into Docker Hub..."
	@if [ -z "$$DOCKER_TOKEN" ]; then echo "❌ DOCKER_TOKEN not set"; exit 1; fi
	echo $$DOCKER_TOKEN | docker login -u $(DOCKER_USER) --password-stdin

push:
	@echo "📤 Pushing $(IMAGE_TAGGED)..."
	docker push $(IMAGE_TAGGED)

pushx: login buildx
	@echo "✅ Multi-arch image pushed to Docker Hub."

# --------------------------------------
# 🚀 Release Version
# --------------------------------------
release:
	@if [ "$(TAG)" = "latest" ]; then echo "❌ You must specify a TAG (e.g., make release TAG=0.1.0)"; exit 1; fi
	@echo "🏷  Releasing version $(TAG)..."
	docker tag $(IMAGE_NAME):latest $(IMAGE_NAME):$(TAG)
	@echo "📤 Pushing both 'latest' and '$(TAG)' tags..."
	docker push $(IMAGE_NAME):latest
	docker push $(IMAGE_NAME):$(TAG)
	@echo "✅ Successfully published: $(IMAGE_NAME):$(TAG)"

# --------------------------------------
# 🧹 Cleanup
# --------------------------------------
clean:
	@echo "🧹 Cleaning local images..."
	-docker rmi $(IMAGE_TAGGED) $(IMAGE_NAME):latest 2>/dev/null || true
	@$(MAKE) clean-go

.PHONY: all init tidy verify build-local run-local clean-go build run builder buildx login push pushx release clean
