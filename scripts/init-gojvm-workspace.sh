#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GOJVM_SOURCE_DIR="${GOJVM_SOURCE_DIR:-$ROOT_DIR/gojvm-source}"
UPSTREAM_REMOTE_URL="${UPSTREAM_REMOTE_URL:-https://go.googlesource.com/go}"
TAG="${GOJVM_UPSTREAM_TAG:-go1.26.4}"

log() {
  printf '[gojvm:init] %s\n' "$*"
}

if [[ -d "$GOJVM_SOURCE_DIR/.git" ]]; then
  log "Using existing source tree: $GOJVM_SOURCE_DIR"
  (cd "$GOJVM_SOURCE_DIR" && git remote -v | grep -q 'upstream')
  if [[ "${1:-}" != "--skip-remote-check" ]]; then
    if ! (cd "$GOJVM_SOURCE_DIR" && git remote get-url upstream >/dev/null 2>&1); then
      log "Creating upstream remote."
      (cd "$GOJVM_SOURCE_DIR" && git remote add upstream "$UPSTREAM_REMOTE_URL")
    fi
  fi
  log "Updating upstream refs."
  (cd "$GOJVM_SOURCE_DIR" && git fetch upstream --tags)
else
  log "Cloning Go upstream into $GOJVM_SOURCE_DIR."
  git clone --depth 1 --branch "$TAG" "$UPSTREAM_REMOTE_URL" "$GOJVM_SOURCE_DIR"
fi

cd "$GOJVM_SOURCE_DIR"
git remote set-url --add --push upstream no_push
if git show-ref --verify --quiet "refs/heads/feature/gojvm-backend"; then
  log "Checkout existing feature/gojvm-backend branch."
  git switch feature/gojvm-backend
else
  log "Creating feature/gojvm-backend from $TAG."
  git switch -c feature/gojvm-backend "$TAG"
fi

log "Workspace ready at $(git rev-parse --short HEAD) on branch $(git branch --show-current)."
log "Add project-local changes under this tree only."

