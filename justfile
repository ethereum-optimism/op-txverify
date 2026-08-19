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
  go build -o dist/op-txverify ./cmd/op-txverify
  @echo "Build completed"

# Build the wasm module for local inspection; the published path is handled elsewhere
build-wasm:
  mkdir -p dist/wasm
  GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o dist/wasm/main.wasm ./cmd/wasm
  @echo "Wasm build completed"
