.PHONY: build install test test-verbose test-coverage clean lint

# Build the binary
build:
	@echo "Building bulletproof..."
	@mkdir -p bin
	@go build -o bin/bulletproof ./cmd/bulletproof

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

# Run linter (requires golangci-lint)
lint:
	@golangci-lint run

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/

# Cross-compile for multiple platforms
build-all:
	@echo "Building for all platforms..."
	@mkdir -p bin
	@GOOS=linux GOARCH=amd64 go build -o bin/bulletproof-linux-amd64 ./cmd/bulletproof
	@GOOS=darwin GOARCH=amd64 go build -o bin/bulletproof-darwin-amd64 ./cmd/bulletproof
	@GOOS=darwin GOARCH=arm64 go build -o bin/bulletproof-darwin-arm64 ./cmd/bulletproof
	@GOOS=windows GOARCH=amd64 go build -o bin/bulletproof-windows-amd64.exe ./cmd/bulletproof
	@echo "Built binaries in bin/"
