.PHONY: build install test test-verbose test-coverage clean lint fmt vet check release

# Version info
VERSION ?= dev
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X github.com/bulletproof-bot/backup/internal/version.Version=$(VERSION) \
           -X github.com/bulletproof-bot/backup/internal/version.GitCommit=$(GIT_COMMIT) \
           -X github.com/bulletproof-bot/backup/internal/version.BuildDate=$(BUILD_DATE)

# Build the binary
build:
	@echo "Building bulletproof..."
	@mkdir -p bin
	@go build -ldflags "$(LDFLAGS)" -o bin/bulletproof ./cmd/bulletproof

# Install to GOPATH/bin
install:
	@echo "Installing bulletproof..."
	@go install ./cmd/bulletproof

# Run all tests
test:
	@go test ./...

# Run tests with verbose output
test-verbose:
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@go test -cover ./...

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Run linter (requires golangci-lint)
lint:
	@echo "Running golangci-lint..."
	@golangci-lint run

# Run all checks (format, vet, lint, test)
check: fmt vet lint test
	@echo "✅ All checks passed"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/

# Cross-compile for multiple platforms
build-all:
	@echo "Building for all platforms..."
	@mkdir -p bin
	@GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/bulletproof-linux-amd64 ./cmd/bulletproof
	@GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/bulletproof-darwin-amd64 ./cmd/bulletproof
	@GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/bulletproof-darwin-arm64 ./cmd/bulletproof
	@GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/bulletproof-windows-amd64.exe ./cmd/bulletproof
	@echo "Built binaries in bin/"

# Create a release snapshot (requires goreleaser)
release:
	@echo "Creating release snapshot..."
	@goreleaser release --snapshot --clean
	@echo "Release artifacts in dist/"
