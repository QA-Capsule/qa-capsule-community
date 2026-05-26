#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DIR="${ROOT_DIR}/tests/playwright"
cd "${DIR}"

npm install
npx playwright install --with-deps chromium
npm run test:ci || true

JUNIT_FILE="${DIR}/playwright-results.xml"
chmod +x "${ROOT_DIR}/tests/upload-junit.sh"
"${ROOT_DIR}/tests/upload-junit.sh" "Playwright" "${JUNIT_FILE}" || true
