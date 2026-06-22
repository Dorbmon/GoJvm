#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fail=0

check_file() {
  local path="$1"
  if [[ ! -f "$ROOT_DIR/$path" ]]; then
    echo "missing: $path"
    fail=1
  fi
}

check_dir() {
  local path="$1"
  if [[ ! -d "$ROOT_DIR/$path" ]]; then
    echo "missing dir: $path"
    fail=1
  fi
}

check_file "goal.md"
check_file "README.md"
check_file ".github/workflows/gojvm-ci.yml"
check_file "docs/gojvm/progress.md"
check_file "docs/gojvm/compatibility.md"
check_file "docs/gojvm/adr/README.md"
check_dir "docs/gojvm/adr"

if [[ "$(git -C "$ROOT_DIR" branch --show-current)" != "feature/gojvm-backend" ]]; then
  echo "expected branch feature/gojvm-backend; local branch is $(git -C "$ROOT_DIR" branch --show-current)."
  fail=1
fi

for f in scripts/*.sh; do
  bash -n "$f"
done

if [[ "$fail" -ne 0 ]]; then
  echo "Milestone check failed."
  exit 1
fi

echo "Milestone check passed."
