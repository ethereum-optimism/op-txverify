# Default recipe to run when just is called without arguments
default:
  @just --list

# Clean the distribution folder
clean:
  rm -rf dist/
  @echo "Cleaned dist/ directory"

# Run tests
test:
  go test ./...
  @echo "Tests completed"

# Run linting
lint:
  golangci-lint run
  @echo "Linting completed"

# Build the project
build:
  mkdir -p dist
  go build -o dist/op-verify ./cmd/op-verify
  @echo "Build completed"

# Run goreleaser in local mode (no publishing)
release-dry-run:
  goreleaser release --snapshot --clean
  @echo "Dry run release completed"

# Release the project using goreleaser
release: clean test lint
  goreleaser release
  @echo "Release completed"
