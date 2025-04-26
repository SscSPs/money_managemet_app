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

# Build the binary
build: clean swagger
	@echo "Building the project..."
	go build -o $(BIN_PATH) ${BASE_ENTRY}

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

release: clean swagger build
	@echo "Release Build Created..."

# Display help message
help:
	@echo "Makefile for MMA_backend"
	@echo ""
	@echo "Usage:"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make swagger        - Generate Swagger documentation"
	@echo "  make build          - Build the project (clean, swagger, then builds)"
	@echo "  make run            - Run unit tests, build, then run the application"
	@echo "  make run-fast       - Build and run the application (NO tests)"
	@echo "  make run-all-tests  - Run all tests (unit+integration), build, then run"
	@echo "  make test           - Run unit tests"
	@echo "  make test-integration - Run integration tests (requires Docker)"
	@echo "  make test-all       - Run both unit and integration tests"
	@echo "  make lint           - Run linter"
	@echo "  make deps           - Install Go modules"
	@echo "  make prod           - Run the app in prod mode (DOES NOT run tests first)"
	@echo "  make release        - Creates the release build"
	@echo "  make help           - Show this help message"

.PHONY: build run run-fast run-all-tests run-swagger clean test test-integration test-all swagger lint deps dev prod help release
