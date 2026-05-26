#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DIR="${ROOT_DIR}/tests/cypress"
cd "${DIR}"

npm install
npm run test:ci || true

JUNIT_FILE="${DIR}/cypress-results.xml"
chmod +x "${ROOT_DIR}/tests/upload-junit.sh"
"${ROOT_DIR}/tests/upload-junit.sh" "Cypress" "${JUNIT_FILE}" || true
