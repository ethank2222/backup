# Makefile for Backup System

# Variables
BINARY_NAME=backup
BUILD_DIR=build

# Go variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GORUN=$(GOCMD) run

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(shell git describe --tags --always --dirty) -X main.BuildTime=$(shell date -u '+%Y-%m-%d_%H:%M:%S')"

# Default target
.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help message
	@echo "Backup System - Available commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the backup binary
	@echo "Building backup binary..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)

.PHONY: test
test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@mkdir -p coverage
	$(GOTEST) -coverprofile=coverage/coverage.out ./...
	$(GOCMD) tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "Coverage report generated: coverage/coverage.html"

.PHONY: run
run: ## Run the backup system
	@echo "Running backup system..."
	$(GORUN) backup.go

.PHONY: install
install: ## Install dependencies
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

.PHONY: vet
vet: ## Vet code
	@echo "Vetting code..."
	$(GOCMD) vet ./...

.PHONY: check
check: fmt vet test ## Run all code quality checks

.PHONY: validate-config
validate-config: ## Validate configuration file
	@echo "Validating configuration..."
	@if [ -f repositories.txt ]; then \
		echo "Repositories file exists"; \
		echo "Repositories to backup:"; \
		grep -v '^#' repositories.txt | grep -v '^$$' || echo "No repositories found"; \
	else \
		echo "Error: repositories.txt not found - this file is required"; \
		exit 1; \
	fi

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t backup-system .

.PHONY: docker-run
docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run --rm -e BACKUP_TOKEN=your-token-here backup-system

.PHONY: release
release: clean build test ## Create release build
	@echo "Creating release..."
	@mkdir -p release
	@cp $(BUILD_DIR)/* release/
	@cp repositories.txt release/
	@cp README-backup.md release/
	@echo "Release files created in release/ directory"

.PHONY: dev
dev: install check run ## Development workflow

.PHONY: verify
verify: check build ## Run all verification steps 