# Strigoi Makefile
# Professional build and development workflow

.PHONY: all build test clean install lint security help

# Variables
BINARY_NAME := strigoi
BINARY_PATH := ./cmd/strigoi
GO := go
GOFLAGS := -v
LDFLAGS := -ldflags "-X main.version=$$(git describe --tags --always --dirty) -X main.build=$$(date -u +%Y%m%d.%H%M%S)"

# Default target
all: clean lint test build

## help: Show this help message
help:
	@echo "Strigoi Development Commands:"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "\033[36m%-15s\033[0m %s\n", "Target", "Description"} /^[a-zA-Z_-]+:.*?##/ { printf "\033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_NAME) $(BINARY_PATH)
	@echo "✓ Build complete: ./$(BINARY_NAME)"

## test: Run all tests
test:
	@echo "Running tests..."
	@$(GO) test $(GOFLAGS) ./...
	@echo "✓ All tests passed"

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	@$(GO) test -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"
	@echo "Coverage summary:"
	@$(GO) tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'

## test-race: Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	@$(GO) test -race ./...
	@echo "✓ No race conditions detected"

## lint: Run linters
lint:
	@echo "Running linters..."
	@if ! which golangci-lint > /dev/null; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@golangci-lint run ./...
	@echo "✓ Linting passed"

## fmt: Format code
fmt:
	@echo "Formatting code..."
	@$(GO) fmt ./...
	@echo "✓ Code formatted"

## security: Run security scan
security:
	@echo "Running security scan..."
	@if ! which gosec > /dev/null; then \
		echo "Installing gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi
	@gosec -quiet ./...
	@echo "✓ Security scan passed"

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	@$(GO) mod download
	@$(GO) mod tidy
	@echo "✓ Dependencies updated"

## deps-check: Check for vulnerabilities in dependencies
deps-check:
	@echo "Checking dependencies for vulnerabilities..."
	@if ! which nancy > /dev/null; then \
		echo "Installing nancy..."; \
		go install github.com/sonatype-nexus-community/nancy@latest; \
	fi
	@go list -json -m all | nancy sleuth
	@echo "✓ No vulnerable dependencies found"

## install: Install the binary
install: build
	@echo "Installing $(BINARY_NAME)..."
	@$(GO) install $(LDFLAGS) $(BINARY_PATH)
	@echo "✓ $(BINARY_NAME) installed to $(GOPATH)/bin"

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out coverage.html
	@rm -rf dist/
	@echo "✓ Clean complete"

## run: Run the application
run: build
	@echo "Starting $(BINARY_NAME)..."
	@./$(BINARY_NAME)

## dev: Run with live reload (requires air)
dev:
	@if ! which air > /dev/null; then \
		echo "Installing air..."; \
		go install github.com/cosmtrek/air@latest; \
	fi
	@air -c .air.toml

## bench: Run benchmarks
bench:
	@echo "Running benchmarks..."
	@$(GO) test -bench=. -benchmem ./...

## proto: Generate protobuf code (if needed)
proto:
	@echo "Generating protobuf code..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		internal/state/*.proto
	@echo "✓ Protobuf generation complete"

## docs: Generate documentation
docs:
	@echo "Generating documentation..."
	@if ! which godoc > /dev/null; then \
		echo "Installing godoc..."; \
		go install golang.org/x/tools/cmd/godoc@latest; \
	fi
	@echo "Documentation server starting at http://localhost:6060"
	@godoc -http=:6060

## release: Create a new release
release: clean lint security test build
	@echo "Creating release..."
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION not specified. Use: make release VERSION=v0.5.0"; \
		exit 1; \
	fi
	@echo "Building release $(VERSION)..."
	@mkdir -p dist
	@GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 $(BINARY_PATH)
	@GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 $(BINARY_PATH)
	@GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe $(BINARY_PATH)
	@echo "✓ Release artifacts created in dist/"

## ci: Run CI pipeline locally
ci: deps lint security test-coverage test-race build
	@echo "✓ CI pipeline passed"

# Git hooks setup
## setup: Setup development environment
setup:
	@echo "Setting up development environment..."
	@$(GO) mod download
	@if ! which pre-commit > /dev/null; then \
		echo "Installing pre-commit..."; \
		pip install pre-commit || echo "Warning: pre-commit installation failed"; \
	fi
	@pre-commit install || echo "Warning: pre-commit hooks not installed"
	@echo "✓ Development environment ready"

# Quick commands for common tasks
## quick-fix: Format and fix common issues
quick-fix: fmt deps
	@echo "✓ Quick fixes applied"

## check: Run all checks without building
check: lint security test
	@echo "✓ All checks passed"