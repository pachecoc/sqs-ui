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
	@if git rev-parse "v$(TAG)" >/dev/null 2>&1; then \
		read -p "⚠️ Git tag v$(TAG) already exists. Overwrite? [y/N]: " confirm; \
		if [ "$$confirm" != "y" ] && [ "$$confirm" != "Y" ]; then \
			echo "❌ Aborting new tag."; \
		fi \
		else \
			@echo "🔖 Creating/updating Git tag v$(TAG)..."
			git tag -d v$(TAG) 2>/dev/null || true
			git tag -a v$(TAG) -m "Release v$(TAG)"
			git push origin :refs/tags/v$(TAG) 2>/dev/null || true
			git push origin v$(TAG)
		fi
	fi

	# --- GitHub Release ---
	@if command -v gh >/dev/null 2>&1; then \
		if gh release view $(TAG) >/dev/null 2>&1; then \
			read -p "⚠️ GitHub release $(TAG) already exists. Overwrite? [y/N]: " confirm; \
			if [ "$$confirm" != "y" ] && [ "$$confirm" != "Y" ]; then \
				echo "❌ Aborting release."; \
			fi \
			else \
				@if command -v gh >/dev/null 2>&1; then \
					echo "🚀 Creating or updating GitHub release v$(TAG)..."; \
					gh release delete $(TAG) --yes 2>/dev/null; \
					if [ -f CHANGELOG.md ]; then \
						gh release create $(TAG) --title "Release $(TAG)" \
							--notes-file CHANGELOG.md || true; \
					else \
						echo "⚠️ No CHANGELOG.md found. Release notes will be generic. Edit the release on GitHub to customize."; \
						gh release create $(TAG) --title "Release $(TAG)" \
							--notes "Automated release for version $(TAG)\nCommit: $(GIT_COMMIT)\nBuilt: $(BUILD_DATE)" || true; \
					fi
				fi
			fi
		fi
	fi

	@echo "✅ Published Docker + Git + GitHub release for $(TAG) (some steps may be skipped if required tools are missing)"

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
