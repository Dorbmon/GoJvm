#!/usr/bin/env bash
set -euo pipefail

JAVA_HOME="${JAVA_HOME:-}"
if [[ -z "$JAVA_HOME" ]]; then
  echo "JAVA_HOME is not set"
  exit 1
fi

mkdir -p build/rt-classes
$JAVA_HOME/bin/javac --release 21 -d build/rt-classes $(find src/main/java -name '*.java')
$JAVA_HOME/bin/jar --create --file build/gojvm-rt.jar -C build/rt-classes .
