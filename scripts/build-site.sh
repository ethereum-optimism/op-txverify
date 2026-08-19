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

rm -rf site
for f in $FILES; do
  mkdir -p "site/$(dirname "$f")"
  cp "core/web/$f" "site/$f"
done

echo "build-site: published $(find site -type f | wc -l | tr -d ' ') files to site/"
