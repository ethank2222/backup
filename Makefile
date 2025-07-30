# Bare-bones Production Makefile for GitHub Repository Backup Tool

# Variables
APP_NAME := backup
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR := build

# Go settings
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GORUN := $(GOCMD) run

# Build flags
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -s -w"

# Default target
.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help message
	@echo "GitHub Repository Backup Tool - Available commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the backup binary
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) backup.go
	@echo "✅ Build complete: $(BUILD_DIR)/$(APP_NAME)"

.PHONY: build-all
build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux backup.go
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows.exe backup.go
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin backup.go
	@echo "✅ Cross-platform builds complete"

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@echo "✅ Clean complete"

.PHONY: test
test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v ./...
	@echo "✅ Tests complete"

.PHONY: run
run: ## Run the backup system
	@echo "Running backup system..."
	@if [ -z "$(BACKUP_TOKEN)" ]; then \
		echo "⚠️  Warning: BACKUP_TOKEN not set"; \
	fi
	$(GORUN) backup.go

.PHONY: validate
validate: ## Validate configuration and environment
	@echo "Validating configuration..."
	@if [ ! -f repositories.txt ]; then \
		echo "❌ Error: repositories.txt not found"; \
		exit 1; \
	fi
	@if [ -z "$(BACKUP_TOKEN)" ]; then \
		echo "❌ Error: BACKUP_TOKEN not set"; \
		exit 1; \
	fi
	@if [ -z "$(GITHUB_REPOSITORY)" ]; then \
		echo "❌ Error: GITHUB_REPOSITORY not set"; \
		exit 1; \
	fi
	@echo "✅ Configuration validated"

.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	$(GOCMD) fmt ./...
	@echo "✅ Code formatted"

.PHONY: vet
vet: ## Vet code
	@echo "Vetting code..."
	$(GOCMD) vet ./...
	@echo "✅ Code vetted"

.PHONY: check
check: fmt vet test ## Run all code quality checks

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t github-backup:latest .
	@echo "✅ Docker image built"

.PHONY: docker-run
docker-run: ## Run Docker container
	@echo "Running Docker container..."
	@if [ -z "$(BACKUP_TOKEN)" ]; then \
		echo "❌ Error: BACKUP_TOKEN not set"; \
		exit 1; \
	fi
	docker run --rm \
		-e BACKUP_TOKEN=$(BACKUP_TOKEN) \
		-e GITHUB_REPOSITORY=$(GITHUB_REPOSITORY) \
		-e WEBHOOK_URL=$(WEBHOOK_URL) \
		github-backup:latest

.PHONY: release
release: clean build-all ## Create release package
	@echo "Creating release package..."
	@mkdir -p release
	@cp $(BUILD_DIR)/* release/
	@cp repositories.txt release/ 2>/dev/null || echo "⚠️  repositories.txt not found"
	@cp README.md release/
	@echo "✅ Release files created in release/"

.PHONY: production
production: validate check build ## Production build with validation
	@echo "✅ Production build complete"

.PHONY: quick
quick: build run ## Quick build and run
	@echo "✅ Quick build and run complete" 