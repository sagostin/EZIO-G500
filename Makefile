# Build and development targets
.PHONY: all build test clean install release dev

# Default target
all: build

# Build for current platform
build:
	go build -o eziolcd ./cmd/eziolcd

# Build for pfSense (FreeBSD amd64)
build-pfsense:
	GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o eziolcd-freebsd-amd64 ./cmd/eziolcd

# Build all platforms
build-all:
	./scripts/build.sh

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run linter
lint:
	go vet ./...
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed, skipping"

# Clean build artifacts
clean:
	rm -rf dist/
	rm -f eziolcd eziolcd-*
	rm -f coverage.out coverage.html

# Install locally
install: build
	cp eziolcd /usr/local/bin/

# Development mode - build and run status
dev:
	go run ./cmd/eziolcd status

# Development mode - run with mock (no hardware)
dev-demo:
	go run ./cmd/eziolcd demo

# Format code
fmt:
	go fmt ./...

# Update dependencies
deps:
	go mod tidy
	go mod download

# Generate a release
release:
	@if [ -z "$(VERSION)" ]; then echo "Usage: make release VERSION=v1.0.0"; exit 1; fi
	git tag $(VERSION)
	git push origin $(VERSION)
	@echo "Tagged $(VERSION) - GitHub Actions will create the release"

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build for current platform"
	@echo "  build-pfsense  - Build for pfSense (FreeBSD amd64)"
	@echo "  build-all      - Build for all platforms"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  lint           - Run linters"
	@echo "  clean          - Remove build artifacts"
	@echo "  install        - Install locally"
	@echo "  dev            - Run status command for development"
	@echo "  dev-demo       - Run demo command for development"
	@echo "  fmt            - Format code"
	@echo "  deps           - Update dependencies"
	@echo "  release        - Create a release (VERSION=v1.0.0)"
