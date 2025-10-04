# --------------------------------------
# 🧱 SQS-UI  |  Local Multi-Arch Build (No Login Needed)
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
	@echo "🔍 Verifying module dependencies..."
	go mod verify

build-local:
	@echo "🔨 Building local binary..."
	CGO_ENABLED=0 go build -o bin/sqs-ui ./cmd/server

run-local:
	@echo "🏃 Running sqs-ui locally..."
	QUEUE_NAME=example go run ./cmd/server

clean-go:
	@echo "🧹 Cleaning Go artifacts..."
	rm -rf bin/ go.sum

# --------------------------------------
# 🐳 Docker Build & Push (Multi-Arch Default)
# --------------------------------------

builder:
	@echo "🧱 Ensuring buildx builder exists..."
	-docker buildx create --name $(BUILDER) --use
	docker buildx inspect --bootstrap

# Build & push multi-arch image directly (no login)
build: builder
	@echo "🚀 Building and pushing multi-arch image to Docker Hub..."
	docker buildx build \
		--platform $(PLATFORMS) \
		-t $(IMAGE_TAGGED) \
		--push .
	@echo "✅ Multi-arch image pushed: $(IMAGE_TAGGED)"

# Push target (alias for build)
push: build
	@echo "✅ Multi-arch image successfully pushed: $(IMAGE_TAGGED)"

# Publish both latest and versioned tags
release:
	@if [ "$(TAG)" = "latest" ]; then echo "❌ You must specify TAG (e.g., make release TAG=0.1.0)"; exit 1; fi
	@echo "🏷️  Releasing version $(TAG)..."
	@$(MAKE) push TAG=$(TAG)
	@echo "📤 Also tagging as latest..."
	docker buildx imagetools create -t $(IMAGE_NAME):latest $(IMAGE_NAME):$(TAG)
	@echo "✅ Published $(IMAGE_NAME):$(TAG) and latest"

# --------------------------------------
# 🔍 Utility Commands
# --------------------------------------

check:
	@echo "🔎 Docker info:"
	@docker info --format '  Username: {{.RegistryConfig.IndexConfigs."docker.io".Name}}' || echo "  Not logged in"
	@echo
	@echo "🧩 Architectures available for $(IMAGE_NAME):"
	@docker buildx imagetools inspect $(IMAGE_NAME):$(TAG) --format '{{json .Manifest.Manifests}}' 2>/dev/null | jq '.[].platform' || echo "  No manifest found."

# --------------------------------------
# 🧹 Cleanup
# --------------------------------------
clean:
	@echo "🧹 Cleaning local Docker images and Go artifacts..."
	-docker rmi $(IMAGE_TAGGED) $(IMAGE_NAME):latest 2>/dev/null || true
	@$(MAKE) clean-go

.PHONY: all init tidy verify build-local run-local clean-go builder build push release check clean
