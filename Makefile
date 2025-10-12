# --------------------------------------
# 🧱 SQS-UI  |  Local Multi-Arch Build (No Login Needed)
# --------------------------------------

DOCKER_USER ?= pachecoc
IMAGE_NAME ?= $(DOCKER_USER)/sqs-ui
TAG ?= latest
PLATFORMS ?= linux/amd64,linux/arm64
BUILDER ?= multiarch-builder
IMAGE_TAGGED := $(IMAGE_NAME):$(TAG)

# --- Commit details
GIT_COMMIT := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X github.com/$(DOCKER_USER)/sqs-ui/internal/version.Version=$(TAG) \
           -X github.com/$(DOCKER_USER)/sqs-ui/internal/version.Commit=$(GIT_COMMIT) \
           -X github.com/$(DOCKER_USER)/sqs-ui/internal/version.BuildTime=$(BUILD_DATE)

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

build-local:
	@echo "🔨 Building local binary with version metadata..."
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/sqs-ui ./cmd/server

version-check: build-local
	@echo "🔍 Checking binary version..."
	./bin/sqs-ui --version

run-local:
	@echo "🏃 Running sqs-ui locally..."
	QUEUE_NAME=$(QUEUE_NAME) go run ./cmd/server

dev:
	@echo "🚀 Running sqs-ui with live reload (Air)..."
	@if ! command -v air >/dev/null 2>&1; then \
		echo "❌ Air is not installed. Run: go install github.com/air-verse/air@latest"; \
		exit 1; \
	fi
	@export $$(grep -v '^#' .env | xargs) && air

clean-go:
	@echo "🧹 Cleaning Go artifacts..."
	rm -rf bin/ go.sum

# --------------------------------------
# 🐳 Docker Build & Push (Multi-Arch Default)
# --------------------------------------

builder:
	@echo "🧱 Ensuring buildx builder exists..."
	@if ! docker buildx inspect $(BUILDER) >/dev/null 2>&1; then \
		docker buildx create --name $(BUILDER) --use; \
	fi
	docker buildx inspect --bootstrap

# Build & push multi-arch image directly (no login)
build: builder
	@echo "🚀 Building and pushing multi-arch image to Docker Hub..."
	docker buildx build \
		--platform $(PLATFORMS) \
		--build-arg VERSION=$(TAG) \
		--build-arg COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_DATE) \
		-t $(IMAGE_TAGGED) \
		--push .
	@echo "✅ Multi-arch image pushed: $(IMAGE_TAGGED)"

# Push target (alias for build)
push: build
	@echo "✅ Multi-arch image successfully pushed: $(IMAGE_TAGGED)"

release:
	@if [ "$(TAG)" = "latest" ]; then \
		echo "❌ You must specify TAG (e.g., make release TAG=0.3.0)"; exit 1; \
	fi
	@echo "🏷️  Releasing version $(TAG)..."

	# --- Build & Push Docker image with version metadata ---
	@$(MAKE) push TAG=$(TAG)

	# --- Tag 'latest' ---
	@echo "📤 Tagging as latest on Docker Hub..."
	@docker buildx imagetools create -t $(IMAGE_NAME):latest $(IMAGE_NAME):$(TAG)

	# --- Git Tag ---
	@echo "🔖 Creating/updating Git tag $(TAG)..."
	-git tag -d $(TAG) 2>/dev/null || true
	git tag -a $(TAG) -m "Release $(TAG)"
	git push origin :refs/tags/$(TAG) 2>/dev/null || true
	git push origin $(TAG)

	# --- GitHub Release ---
	@if command -v gh >/dev/null 2>&1; then \
		echo "🚀 Creating or updating GitHub release $(TAG)..."; \
		gh release delete $(TAG) --yes 2>/dev/null || true; \
		gh release create $(TAG) --title "Release $(TAG)" \
			--notes "$$(printf '%s\n%s\n\n🧱 Docker Image:\n```bash\ndocker pull %s:%s\ndocker run --rm %s:%s --version\n```\n\nMulti-arch: linux/amd64, linux/arm64' \
			'Automated release for version $(TAG)' \
			'Built Date: $(BUILD_DATE)' \
			'$(IMAGE_NAME)' '$(TAG)' '$(IMAGE_NAME)' '$(TAG)')" || true; \
	else \
		echo "⚠️ GitHub CLI (gh) not installed — skipping GitHub release."; \
	fi

	@echo "✅ Published Docker + Git + GitHub release for $(TAG)"

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
