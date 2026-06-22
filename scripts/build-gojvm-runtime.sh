#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUNTIME_DIR="$ROOT_DIR/src/gojvm/runtime-java"
JAVA_HOME="${JAVA_HOME:-}"

if [[ -n "${GOJVM_SKIP_RUNTIME_CHECK:-}" ]]; then
  echo "runtime check skipped by GOJVM_SKIP_RUNTIME_CHECK"
  exit 0
fi

if [[ -z "$JAVA_HOME" ]]; then
  if command -v java >/dev/null 2>&1; then
    JAVA_BIN="$(command -v java)"
    JAVA_HOME="$(dirname "$(dirname "$JAVA_BIN")")"
  else
    echo "gojvm: JAVA_HOME is not set and javac was not found on PATH"
    exit 1
  fi
fi

JAVAC="$JAVA_HOME/bin/javac"
JAR="$JAVA_HOME/bin/jar"
if [[ ! -x "$JAVAC" ]]; then
  echo "gojvm: javac not executable at $JAVAC"
  exit 1
fi
if [[ ! -x "$JAR" ]]; then
  echo "gojvm: jar not executable at $JAR"
  exit 1
fi

BUILD_DIR="$RUNTIME_DIR/build"
CLASSES_DIR="$BUILD_DIR/rt-classes"
JAR_PATH="$BUILD_DIR/gojvm-rt.jar"

rm -rf "$BUILD_DIR"
mkdir -p "$CLASSES_DIR"

JAVA_SOURCES=( "$RUNTIME_DIR"/src/main/java/org/gojvm/rt/*.java )
if [[ "${#JAVA_SOURCES[@]}" -eq 1 && "${JAVA_SOURCES[0]}" == "$RUNTIME_DIR/src/main/java/org/gojvm/rt/*.java" ]]; then
  echo "gojvm: no Java source files found under $RUNTIME_DIR/src/main/java"
  exit 1
fi

"$JAVAC" --release 21 -d "$CLASSES_DIR" "${JAVA_SOURCES[@]}"
"$JAR" --create --file "$JAR_PATH" --manifest=/dev/stdin -C "$CLASSES_DIR" . <<EOF
Manifest-Version: 1.0
Main-Class: org.gojvm.rt.RuntimeBootstrap
EOF

if ! "$JAR" tf "$JAR_PATH" | grep -q 'org/gojvm/rt/RuntimeBootstrap.class'; then
  echo "gojvm: expected RuntimeBootstrap.class missing from runtime jar"
  exit 1
fi

echo "gojvm-runtime jar built at $JAR_PATH"
