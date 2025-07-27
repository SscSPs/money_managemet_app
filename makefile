# Makefile for mma_backend

# Define default target
.DEFAULT_GOAL := help

# Variables
APP_NAME := mma_backend
BIN_DIR := bin
BIN_PATH := $(BIN_DIR)/$(APP_NAME)
GO_FILES := $(shell find . -name '*.go' -not -path './vendor/*')
BASE_ENTRY := ./cmd/mma_backend

# Clean the build artifacts
clean:
	@echo "Cleaning up..."
	rm -f $(BIN_PATH)

# Generate Swagger documentation
swagger:
	@echo "Generating Swagger documentation..."
	swag init -g ${BASE_ENTRY}/main.go -o cmd/docs

# Build the binary (dynamic)
build: clean swagger
	@echo "Building the project..."
	go build -o $(BIN_PATH) ${BASE_ENTRY}

# Build a statically linked binary (for Docker scratch or distroless)
build-static: clean swagger
	@echo "Building static binary for Linux AMD64..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o $(BIN_PATH) ${BASE_ENTRY}

# Build a statically linked binary for ARM64 (for Docker scratch or distroless)
build-static-arm64: clean swagger
	@echo "Building static binary for Linux ARM64..."
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o $(BIN_PATH)-arm64 ${BASE_ENTRY}

# Run the application with default settings after passing unit tests
run: test build
	@echo "Running the application(dev mode) after unit tests..."
	$(BIN_PATH)

# Run the application WITHOUT running tests first
run-fast: build
	@echo "Running the application(dev mode) without tests..."
	$(BIN_PATH)

# Run the application after passing ALL tests
run-all-tests: test-all build
	@echo "Running the application(dev mode) after all tests..."
	$(BIN_PATH)

# Test the application (Unit Tests Only)
test:
	@echo "Running unit tests..."
	go test ./...

# Run Integration Tests (Requires Docker)
test-integration:
	@echo "Running integration tests (ensure Docker is running)..."
	go test -tags=integration ./...

# Run All Tests (Unit + Integration)
test-all: test test-integration
	@echo "All tests completed."

# Lint the code
lint:
	@echo "Running linter..."
	golangci-lint run

# Install Go modules
deps:
	@echo "Installing Go modules..."
	go mod tidy

# Run the application in production mode
prod:
	@echo "Running in production mode..."
	IS_PRODUCTION=true $(BIN_PATH)

# Create release build (dynamic)
release: clean swagger build
	@echo "Release Build Created..."

# Create release build for Docker (static)
release-static: clean swagger build-static
	@echo "Static Release Build Created..."

# Create release build for Docker ARM64 (static)
release-static-arm64: clean swagger build-static-arm64
	@echo "Static ARM64 Release Build Created..."

# Display help message
help:
	@echo "Makefile for MMA_backend"
	@echo ""
	@echo "Usage:"
	@echo "  make clean            - Clean build artifacts"
	@echo "  make swagger          - Generate Swagger documentation"
	@echo "  make build            - Build the project (dynamic binary)"
	@echo "  make build-static     - Build statically linked binary for Docker scratch/distroless (AMD64)"
	@echo "  make build-static-arm64 - Build statically linked binary for Docker scratch/distroless (ARM64)"
	@echo "  make run              - Run unit tests, build, then run the application"
	@echo "  make run-fast         - Build and run the application (NO tests)"
	@echo "  make run-all-tests    - Run all tests (unit+integration), build, then run"
	@echo "  make test             - Run unit tests"
	@echo "  make test-integration - Run integration tests (requires Docker)"
	@echo "  make test-all         - Run both unit and integration tests"
	@echo "  make lint             - Run linter"
	@echo "  make deps             - Install Go modules"
	@echo "  make prod             - Run the app in prod mode (DOES NOT run tests first)"
	@echo "  make release          - Creates the release build (dynamic)"
	@echo "  make release-static   - Creates a static release build for Docker (AMD64)"
	@echo "  make release-static-arm64 - Creates a static release build for Docker (ARM64)"
	@echo "  make help             - Show this help message"

.PHONY: build build-static build-static-arm64 run run-fast run-all-tests clean test test-integration test-all swagger lint deps prod release release-static release-static-arm64 help
