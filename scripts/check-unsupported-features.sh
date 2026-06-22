#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="${1:-.}"
ROOT_DIR="$(cd "$ROOT_DIR" && pwd)"

if [[ -d "$ROOT_DIR/.git" ]]; then
  ROOT_DIR="$(cd "$ROOT_DIR" && pwd)"
fi

readarray -t GO_FILES < <(find "$ROOT_DIR" -type f -name '*.go' \
  -not -path "*/.git/*" \
  -not -path "*/vendor/*")

if [[ ${#GO_FILES[@]} -eq 0 ]]; then
  echo "No .go files to check."
  exit 0
fi

status=0
for file in "${GO_FILES[@]}"; do
  if grep -nE '^\s*import\s+`?\"C\"`?' "$file" >/dev/null 2>&1; then
    echo "Unsupported import in $file: import \"C\""
    status=1
  fi
  if grep -nE '^\s*import\s*\(\s*' "$file" >/dev/null 2>&1; then
    # detect unsafe in import blocks
    if awk '/^\s*import\s*\(/,/^\)/' "$file" | grep -nE '^\s*\"unsafe\"' >/dev/null 2>&1; then
      echo "Unsupported import in $file: unsafe"
      status=1
    fi
  fi
done

exit $status
