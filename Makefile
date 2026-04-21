.PHONY: build test clean fmt lint install run check serve

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOFMT=$(GOCMD) fmt
GOMOD=$(GOCMD) mod
BINARY_NAME=cert-monitor
BINARY_DIR=bin
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

# Default target
all: build

# Build the binary for the current platform
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME) ./cmd/cert-monitor

# Build for multiple platforms
build-all: build-darwin-arm64 build-darwin-amd64 build-linux-amd64 build-windows-amd64

build-darwin-arm64:
	@echo "Building darwin/arm64..."
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/cert-monitor

build-darwin-amd64:
	@echo "Building darwin/amd64..."
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/cert-monitor

build-linux-amd64:
	@echo "Building linux/amd64..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/cert-monitor

build-windows-amd64:
	@echo "Building windows/amd64..."
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/cert-monitor

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

# Run tests with coverage report
test-cover:
	$(GOTEST) -v -race -coverprofile=coverage.txt ./...
	$(GOCMD) tool cover -html=coverage.txt -o coverage.html

# Format code
fmt:
	$(GOFMT) ./...

# Tidy dependencies
tidy:
	$(GOMOD) tidy
	$(GOMOD) download

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BINARY_DIR)/
	rm -f coverage.txt coverage.html

# Install to PATH
install:
	$(GOCMD) install $(LDFLAGS) ./cmd/cert-monitor

# Run locally (without installing)
run: build
	./$(BINARY_DIR)/$(BINARY_NAME) check google.com:443 github.com:443

# One-shot check example
check: build
	./$(BINARY_DIR)/$(BINARY_NAME) check --config config/config.yaml.example

# Start daemon example
serve: build
	./$(BINARY_DIR)/$(BINARY_NAME) serve --config config/config.yaml.example --addr :8080

# Lint (requires golangci-lint)
lint:
	golangci-lint run ./...

# Generate sample config
gen-config: build
	./$(BINARY_DIR)/$(BINARY_NAME) gen-config --output config/config.yaml.example

# Vet code
vet:
	$(GOCMD) vet ./...
