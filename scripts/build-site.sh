#!/usr/bin/env bash
# Assembles the published site from an explicit file-level allowlist.
# Deliberately NOT published: reader.html (the CLI serves it locally from its
# go:embed copy), lib/qrscanner/, and the lib/*/README.md files.
set -euo pipefail

cd "$(dirname "$0")/.."

# Paths are relative to core/web/ and are preserved under site/: the page loads
# them by root-absolute paths (e.g. /lib/qrcode/qrcode.min.js).
FILES="
index.html
app.css
app.js
verify.js
lib/qrcode/qrcode.min.js
lib/fflate/index.min.js
lib/erasure/erasure.js
lib/sunny.png
"

# Check everything before copying, so a rename fails the build loudly instead of
# silently publishing a smaller site.
missing=0
for f in $FILES; do
  if [ ! -f "core/web/$f" ]; then
    echo "build-site: missing required file: core/web/$f" >&2
    missing=1
  fi
done
if [ "$missing" -ne 0 ]; then
  echo "build-site: allowlist out of sync with core/web/ - fix the allowlist or restore the file(s)" >&2
  exit 1
fi

# Netlify's Drawer and its snippet injection both insert a <script> immediately before the closing
# body tag, and a same-origin script is indistinguishable from ours to `script-src 'self'`. Leaving
# the tag out is the only control this repo holds over that, so it must not come back by accident.
for f in $FILES; do
  case "$f" in
  *.html)
    if grep -qi '</body>\|</html>' "core/web/$f"; then
      echo "build-site: core/web/$f closes body or html, which is where Netlify injects a script; leave both tags off" >&2
      exit 1
    fi
    ;;
  esac
done

rm -rf site
for f in $FILES; do
  mkdir -p "site/$(dirname "$f")"
  cp "core/web/$f" "site/$f"
done

mkdir -p site/wasm
GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o site/wasm/main.wasm ./cmd/wasm

# wasm_exec.js moved from misc/wasm/ to lib/wasm/ in Go 1.24, so both are checked. It is copied from
# GOROOT rather than vendored into core/web/lib/, because core/scan.go's `go:embed web/lib/*` would
# bake the ~7MB blob into every CLI binary.
wasm_exec=""
for candidate in "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$(go env GOROOT)/misc/wasm/wasm_exec.js"; do
  if [ -f "$candidate" ]; then
    wasm_exec="$candidate"
    break
  fi
done
if [ -z "$wasm_exec" ]; then
  echo "build-site: wasm_exec.js not found in $(go env GOROOT) under lib/wasm/ or misc/wasm/" >&2
  exit 1
fi
cp "$wasm_exec" site/wasm/wasm_exec.js
echo "build-site: copied wasm_exec.js from $wasm_exec"

if command -v shasum >/dev/null 2>&1; then
  wasm_sha=$(shasum -a 256 site/wasm/main.wasm | cut -d' ' -f1)
elif command -v sha256sum >/dev/null 2>&1; then
  wasm_sha=$(sha256sum site/wasm/main.wasm | cut -d' ' -f1)
else
  echo "build-site: neither shasum nor sha256sum available" >&2
  exit 1
fi

# A version indicator, not an integrity check: Netlify injects snippets at serve time without ever
# touching the deploy artifact, so this digest would still match while altered HTML was served.
commit="${COMMIT_REF:-$(git rev-parse HEAD)}"
printf '{"commit": "%s", "wasm_sha256": "%s"}\n' "$commit" "$wasm_sha" > site/build-info.json

# A signer waits for this over cellular mid-ceremony, so meaningful growth should fail the build
# rather than go unnoticed. Currently ~2.56MB gzipped; the ceiling allows roughly 25% headroom for
# ordinary dependency drift. Raise it deliberately, with a reason.
GZIP_CEILING=3200000
gzipped=$(gzip -9 -c site/wasm/main.wasm | wc -c | tr -d ' ')
echo "build-site: main.wasm is $(wc -c < site/wasm/main.wasm | tr -d ' ') bytes raw, $gzipped bytes gzipped"
if [ "$gzipped" -gt "$GZIP_CEILING" ]; then
  echo "build-site: main.wasm is $gzipped bytes gzipped, over the $GZIP_CEILING byte ceiling" >&2
  exit 1
fi

echo "build-site: published $(find site -type f | wc -l | tr -d ' ') files to site/"
