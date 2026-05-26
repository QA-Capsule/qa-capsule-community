#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DIR="${ROOT_DIR}/tests/newman"
cd "${DIR}"

npx newman run collection.json \
  --reporters cli,junit \
  --reporter-junit-export newman-results.xml || true

JUNIT_FILE="${DIR}/newman-results.xml"
chmod +x "${ROOT_DIR}/tests/upload-junit.sh"
"${ROOT_DIR}/tests/upload-junit.sh" "Postman" "${JUNIT_FILE}" || true
