# gohatch - Task Runner

# Version from git
version := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`

# Default: show available commands
default:
    @just --list

# Build binary
build:
    go build -ldflags="-s -w -X 'main.version={{version}}'" -trimpath -o build/gohatch ./cmd/gohatch

# Run tests
test *ARGS:
    go test ./... {{ARGS}}

# Run tests with coverage report
cover:
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out

# Format code
fmt:
    go fmt ./...

# Lint code
lint:
    golangci-lint run

# Run all checks (format, lint, test)
check:
    just fmt
    just lint
    just test

# Clean build artifacts
clean:
    rm -rf build/

# Install binary to $GOPATH/bin
install:
    go install -ldflags="-s -w -X 'main.version={{version}}'" ./cmd/gohatch

# Tidy dependencies
tidy:
    go mod tidy

# Build a local snapshot for testing
release-snapshot *ARGS:
    goreleaser release --clean --snapshot {{ARGS}}

# Create and publish a release (requires git tag)
release *ARGS:
    goreleaser release --clean {{ARGS}}
