# ==========================================================
# Makefile for SQS-UI
# ==========================================================
# This Makefile provides build, run, clean, and release targets.
# Comments added for clarity and readability.

# Variables
BUILDER ?= sqs-ui-builder
IMAGE_NAME ?= pachecoc/sqs-ui
TAG ?= $(shell git describe --tags --always)
GIT_COMMIT ?= $(shell git rev-parse HEAD)
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# ------------------------------------------
# Build & Run Targets
# ------------------------------------------

# Keep module files clean before building.
tidy:
	@echo "Running go mod tidy..."
	go mod tidy

build-local: tidy
	@echo "Building sqs-ui locally..."
	go build -v -o sqs-ui ./cmd/server

run-local:
	@echo "Running sqs-ui locally..."
	go run ./cmd/server

dev:
	@echo "Running sqs-ui with live reload (Air)..."
	@if ! command -v air > /dev/null 2>&1; then \
		echo "Air is not installed. Run: go install github.com/air-verse/air@latest"; \
		exit 1; \
	fi
	@export $$(grep -v '^#' .env | xargs) && air

# ------------------------------------------
# Clean Targets
# ------------------------------------------
clean-go:
	@echo "Cleaning Go artifacts..."
	rm -rf sqs-ui *.test *.out *.coverprofile coverage.* profile.cov bin/ build/ dist/ tmp/

# ------------------------------------------
# Docker Build Targets
# ------------------------------------------

builder:
	@echo "Ensuring buildx builder exists..."
	@if ! docker buildx inspect $(BUILDER) > /dev/null 2>&1; then \
		docker buildx create --name $(BUILDER) --use; \
	fi
	docker buildx inspect --bootstrap

# Build & push multi-arch image directly (no login)
buildx:
	make builder
	docker buildx build --platform linux/amd64,linux/arm64 \
		-t $(IMAGE_NAME):$(TAG) \
		-t $(IMAGE_NAME):latest \
		--push .

# ------------------------------------------
# Release Targets
# ------------------------------------------

release:
	make buildx
	@docker buildx imagetools create -t $(IMAGE_NAME):latest $(IMAGE_NAME):$(TAG)
	@echo "Creating/updating Git tag $(TAG)..."
	-git tag -d $(TAG) 2>/dev/null || true
	git tag -a $(TAG) -m "Release $(TAG)"
	git push origin :refs/tags/$(TAG) 2>/dev/null || true
	git push origin $(TAG)
	@if command -v gh > /dev/null 2>&1; then \
		echo "Creating or updating GitHub release $(TAG)..."; \
		gh release delete $(TAG) --yes 2>/dev/null || true; \
		gh release create $(TAG) --title "Release $(TAG)" \
			--notes "$$(printf '%s\n%s\n\nDocker Image:\n```bash\ndocker pull %s:%s\ndocker run --rm %s:%s --version\n```\n\nMulti-arch: linux/amd64, linux/arm64' \
			'Automated release for version $(TAG)' \
			'Built Date: $(BUILD_DATE)' \
			'$(IMAGE_NAME)' '$(TAG)' '$(IMAGE_NAME)' '$(TAG)')" || true; \
	else \
		echo "GitHub CLI (gh) not installed — skipping GitHub release."; \
	fi
	@echo "Published Docker + Git + GitHub release for $(TAG)"

# ------------------------------------------
# Utility Commands
# ------------------------------------------

# Suggestion: Consider adding linting and testing targets for Go and frontend
# lint:
# 	golangci-lint run ./...
# test:
# 	go test ./...

# End of Makefile
