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

# Run GoReleaser in local mode (no publishing)
release-dry-run:
  goreleaser release --snapshot --clean --skip=publish
  version="$(basename dist/op-txverify_*_SHA256SUMS)"; version="${version#op-txverify_}"; version="${version%_SHA256SUMS}"; just _stage-release-assets "$version"
  version="$(basename dist/op-txverify_*_SHA256SUMS)"; version="${version#op-txverify_}"; version="${version%_SHA256SUMS}"; cd dist && shasum -a 256 --check --ignore-missing "op-txverify_${version}_SHA256SUMS"
  @echo "Dry run release completed"

# Build release artifacts and create a draft GitHub release
release tag: clean test lint
  test "{{tag}}" = "$(git describe --tags --exact-match)" || (echo "Current commit is not tagged {{tag}}" >&2; exit 1)
  goreleaser release --clean --skip=publish
  version="{{tag}}"; version="${version#v}"; just _stage-release-assets "$version"
  version="{{tag}}"; version="${version#v}"; cd dist && shasum -a 256 --check --ignore-missing "op-txverify_${version}_SHA256SUMS"
  version="{{tag}}"; version="${version#v}"; gh release create "{{tag}}" "dist/op-txverify_${version}_darwin_amd64" "dist/op-txverify_${version}_darwin_arm64" "dist/op-txverify_${version}_linux_amd64" "dist/op-txverify_${version}_linux_arm64" "dist/op-txverify_${version}_SHA256SUMS" "dist/op-txverify_${version}_source.zip" --draft --title "{{tag}}" --generate-notes --verify-tag
  @echo "Draft release created for {{tag}}"

# Stage GoReleaser binary outputs under their release asset names
_stage-release-assets version:
  cp dist/op-txverify_darwin_amd64_v1/op-txverify "dist/op-txverify_{{version}}_darwin_amd64"
  cp dist/op-txverify_darwin_arm64_v8.0/op-txverify "dist/op-txverify_{{version}}_darwin_arm64"
  cp dist/op-txverify_linux_amd64_v1/op-txverify "dist/op-txverify_{{version}}_linux_amd64"
  cp dist/op-txverify_linux_arm64_v8.0/op-txverify "dist/op-txverify_{{version}}_linux_arm64"
