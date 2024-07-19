# Git
GIT_REVISION ?= $(shell git rev-parse --short HEAD)
GIT_TAG ?= $(shell git describe --tags --abbrev=0 --always | sed -e s/v//g)

# Go
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
GOBUILD ?= GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build
LDFLAGS ?= '-s -w \
	-X "github.com/ks6088ts-labs/misctl/internal.Revision=$(GIT_REVISION)" \
	-X "github.com/ks6088ts-labs/misctl/internal.Version=$(GIT_TAG)" \
'

# Docker
DOCKER_REPO_NAME ?= ks6088ts
DOCKER_IMAGE_NAME ?= template-go
DOCKER_COMMAND ?= /app

# Tools
TOOLS_DIR ?= $(shell go env GOPATH)/bin
GOLANGCI_LINT_VERSION ?= 1.56.2
TRIVY_VERSION ?= 0.49.1

# Misc
OUTPUT_DIR ?= dist
OUTPUT ?= $(OUTPUT_DIR)/template-go

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
.DEFAULT_GOAL := help

.PHONY: install-deps-dev
install-deps-dev: ## install dependencies for development

.PHONY: format-check
format-check: ## format check
	@# https://stackoverflow.com/a/67962664
	test -z "$$(gofmt -e -l -s .)"

.PHONY: format
format: ## format code
	gofmt -e -l -s -w .

.PHONY: lint
lint: ## lint
	@# https://golangci-lint.run/welcome/install/#local-installation
	@which golangci-lint || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(TOOLS_DIR) v$(GOLANGCI_LINT_VERSION)
	golangci-lint run -v

.PHONY: test
test: ## run tests
	go test -cover -v ./...

.PHONY: build
build: ## build applications
	mkdir -p $(OUTPUT_DIR)
	$(GOBUILD) -ldflags=$(LDFLAGS) -o $(OUTPUT) .

.PHONY: ci-test
ci-test: install-deps-dev format-check lint test build ## run CI test

.PHONY: release
release: ## release applications
	@# https://goreleaser.com/install/
	@which goreleaser || go install github.com/goreleaser/goreleaser@latest
	goreleaser release --snapshot --clean

# ---
# Docker
# ---

.PHONY: docker-build
docker-build: ## build Docker image
	docker build \
		-t $(DOCKER_REPO_NAME)/$(DOCKER_IMAGE_NAME):$(GIT_TAG) \
		--build-arg GIT_REVISION=$(GIT_REVISION) \
		--build-arg GIT_TAG=$(GIT_TAG) \
		.

.PHONY: docker-run
docker-run: ## run Docker container
	docker run --rm $(DOCKER_REPO_NAME)/$(DOCKER_IMAGE_NAME):$(GIT_TAG) $(DOCKER_COMMAND)

.PHONY: docker-lint
docker-lint: ## lint Dockerfile
	docker run --rm -i hadolint/hadolint < Dockerfile

.PHONY: docker-scan
docker-scan: ## scan Docker image
	@# https://aquasecurity.github.io/trivy/v0.18.3/installation/#install-script
	@which trivy || curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b $(TOOLS_DIR) v$(TRIVY_VERSION)
	trivy image $(DOCKER_REPO_NAME)/$(DOCKER_IMAGE_NAME):$(GIT_TAG)

.PHONY: ci-test-docker
ci-test-docker: docker-lint docker-build docker-scan docker-run ## run CI test for Docker
