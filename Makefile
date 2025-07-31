# Build configuration
BINARY_NAME=ado-gh-wi-migrator
BUILD_DIR=build
MAIN_PATH=./cmd/migrate

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Version info
VERSION?=dev
COMMIT?=$(shell git rev-parse --short HEAD)
BUILD_TIME?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS=-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)

.PHONY: all build clean test deps tidy help install run-dry run-validate

# Default target
all: clean deps build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
build-all: clean deps
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# Windows
	GOOS=windows GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	
	# Linux
	GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	
	# macOS
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	
	@echo "✓ Multi-platform build complete"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f migration_checkpoint.json
	rm -rf reports/*.json
	@echo "✓ Clean complete"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@echo "✓ Tests complete"

# Test with coverage report
test-coverage: test
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) verify
	@echo "✓ Dependencies downloaded"

# Tidy up go.mod
tidy:
	@echo "Tidying go.mod..."
	$(GOMOD) tidy
	@echo "✓ go.mod tidied"

# Install the application
install: build
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "✓ Installed to $(GOPATH)/bin/$(BINARY_NAME)"

# Development commands

# Initialize configuration
init-config: build
	@echo "Initializing configuration..."
	./$(BUILD_DIR)/$(BINARY_NAME) config init
	@echo "✓ Configuration initialized"

# Validate configuration and connections
validate: build
	@echo "Validating configuration..."
	./$(BUILD_DIR)/$(BINARY_NAME) validate
	@echo "✓ Validation complete"

# Run dry-run migration
run-dry: build
	@echo "Running dry-run migration..."
	./$(BUILD_DIR)/$(BINARY_NAME) migrate --dry-run --verbose

# Run actual migration
run: build
	@echo "Running migration..."
	./$(BUILD_DIR)/$(BINARY_NAME) migrate --verbose

# Run with resume
run-resume: build
	@echo "Resuming migration..."
	./$(BUILD_DIR)/$(BINARY_NAME) migrate --resume --verbose

# Development tools

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...
	@echo "✓ Code formatted"

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	golangci-lint run ./...
	@echo "✓ Linting complete"

# Generate documentation
docs:
	@echo "Generating documentation..."
	$(GOCMD) doc -all ./... > docs/api.txt
	@echo "✓ Documentation generated"

# Security scan (requires gosec)
security:
	@echo "Running security scan..."
	gosec ./...
	@echo "✓ Security scan complete"

# Docker targets (if Docker support is added later)
docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) .
	@echo "✓ Docker image built"

# Release preparation
release: clean deps test build-all
	@echo "Preparing release $(VERSION)..."
	@mkdir -p $(BUILD_DIR)/release
	@cp README.md $(BUILD_DIR)/release/
	@cp configs/config.yaml $(BUILD_DIR)/release/config.example.yaml
	@cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-$(VERSION)-windows-amd64.tar.gz $(BINARY_NAME)-windows-amd64.exe
	@cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	@cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	@cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	@echo "✓ Release $(VERSION) prepared in $(BUILD_DIR)/release/"

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build the application"
	@echo "  build-all     - Build for multiple platforms"
	@echo "  clean         - Clean build artifacts"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  deps          - Download dependencies"
	@echo "  tidy          - Tidy go.mod"
	@echo "  install       - Install the application"
	@echo "  init-config   - Initialize configuration file"
	@echo "  validate      - Validate configuration and connections"
	@echo "  run-dry       - Run dry-run migration"
	@echo "  run           - Run actual migration"
	@echo "  run-resume    - Resume migration from checkpoint"
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code (requires golangci-lint)"
	@echo "  docs          - Generate documentation"
	@echo "  security      - Run security scan (requires gosec)"
	@echo "  release       - Prepare release packages"
	@echo "  help          - Show this help message"
