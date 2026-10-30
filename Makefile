.DEFAULT_GOAL := help

APP_NAME     := engage-engine
BIN_DIR      := bin
API_BIN      := $(BIN_DIR)/api
WORKER_BIN   := $(BIN_DIR)/worker
API_ENTRY    := cmd/api/main.go
WORKER_ENTRY := cmd/worker/main.go

.PHONY: all
all: clean fmt vet test build ## Format, vet, test, and build all binaries

.PHONY: help
help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: build-api build-worker ## Build all service binaries

.PHONY: build-api
build-api: ## Build the API server binary
	@mkdir -p $(BIN_DIR)
	@echo "Building API server..."
	go build -o $(API_BIN) $(API_ENTRY)

.PHONY: build-worker
build-worker: ## Build the worker service binary
	@mkdir -p $(BIN_DIR)
	@echo "Building Worker service..."
	go build -o $(WORKER_BIN) $(WORKER_ENTRY)

.PHONY: run-api
run-api: ## Run the HTTP API server directly
	go run $(API_ENTRY)

.PHONY: run-worker
run-worker: ## Run the background worker service directly
	go run $(WORKER_ENTRY)

.PHONY: test
test: ## Run unit tests with race detection
	go test -v -race ./...

.PHONY: test-coverage
test-coverage: ## Run unit tests and generate coverage report
	@mkdir -p $(BIN_DIR)
	go test -v -race -coverprofile=$(BIN_DIR)/coverage.out ./...
	go tool cover -html=$(BIN_DIR)/coverage.out -o $(BIN_DIR)/coverage.html
	@echo "Coverage report generated at $(BIN_DIR)/coverage.html"

.PHONY: fmt
fmt: ## Format all Go source files
	go fmt ./...

.PHONY: vet
vet: ## Run go vet static analysis
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint if installed
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run || echo "golangci-lint is not installed. Visit https://golangci-lint.run/welcome/install/"

.PHONY: clean
clean: ## Clean build artifacts and coverage files
	rm -rf $(BIN_DIR)

.PHONY: docker-build
docker-build: ## Build Docker images for API and worker
	docker build -f deployments/docker/Dockerfile.api -t $(APP_NAME)-api:latest .
	docker build -f deployments/docker/Dockerfile.worker -t $(APP_NAME)-worker:latest .

.PHONY: docker-up
docker-up: ## Start infrastructure and services with docker-compose
	docker compose -f deployments/docker-compose.yml up -d

.PHONY: docker-down
docker-down: ## Stop and remove docker-compose containers
	docker compose -f deployments/docker-compose.yml down

