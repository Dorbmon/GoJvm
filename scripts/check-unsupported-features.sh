#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="${1:-.}"
ROOT_DIR="$(cd "$ROOT_DIR" && pwd)"
BUILD_MODE="${2:-exe}"

go run ./src/cmd/compile/internal/gojvm/checks/cmd/gojvmcheck \
  -root "$ROOT_DIR" \
  -buildmode "$BUILD_MODE" >/dev/null
exit $?
