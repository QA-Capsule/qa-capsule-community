#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DIR="${ROOT_DIR}/tests/junit-java"
cd "${DIR}"

JAR="${DIR}/junit-standalone.jar"
if [[ ! -f "${JAR}" ]]; then
  echo "Downloading JUnit Platform Console..."
  curl -sL https://repo1.maven.org/maven2/org/junit/platform/junit-platform-console-standalone/1.10.2/junit-platform-console-standalone-1.10.2.jar \
    -o "${JAR}"
fi

mkdir -p classes reports
javac -cp "${JAR}" src/*.java -d classes/

set +e
java -jar "${JAR}" \
  --scan-classpath \
  --class-path classes/ \
  --reports-dir reports/
EXIT=$?
set -e

REPORT="$(find reports -name 'TEST-*.xml' -type f | head -1)"
if [[ -n "${REPORT}" ]]; then
  chmod +x "${ROOT_DIR}/tests/upload-junit.sh"
  "${ROOT_DIR}/tests/upload-junit.sh" "JUnit" "${REPORT}" || true
fi

exit "${EXIT}"
