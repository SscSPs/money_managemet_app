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
build:
	@echo "Building the project..."
	go build -o $(BIN_PATH) ${BASE_ENTRY}

# Run the application with default settings
run: clean swagger build
	@echo "Running the application(dev mode)..."
	$(BIN_PATH)

# Test the application
test: clean swagger build
	@echo "Running tests..."
	go test ./...

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
	@echo "  make clean        - Clean build artifacts"
	@echo "  make swagger      - Generate Swagger documentation"
	@echo "  make build        - Build the project(clean, swagger, then builds)"
	@echo "  make run          - Run the application"
	@echo "  make test         - Run tests"
	@echo "  make lint         - Run linter"
	@echo "  make deps         - Install Go modules"
	@echo "  make prod         - Run the app in prod mode"
	@echo "  make release      - Creates the release build"
	@echo "  make help         - Show this help message"

.PHONY: build run run-swagger clean test swagger lint deps dev prod help
