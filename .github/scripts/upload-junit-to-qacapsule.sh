#!/usr/bin/env bash
# Upload a JUnit XML report to QA Capsule (skips gracefully when secrets or file are missing).
set -euo pipefail

FILE="${1:-}"
FRAMEWORK="${2:-JUnit}"
RUN_ID="${3:-local}"

WEBHOOK_URL="${WEBHOOK_URL:-${QA_CAPSULE_URL:-}}"
API_KEY="${API_KEY:-}"

if [ -z "$WEBHOOK_URL" ] || [ -z "$API_KEY" ]; then
  echo "Skipping QA Capsule upload — set QA_CAPSULE_URL and the project API key secret."
  exit 0
fi

if [ -z "$FILE" ] || [ ! -f "$FILE" ]; then
  echo "JUnit report not found: ${FILE:-<unset>}"
  exit 0
fi

HTTP_STATUS=$(curl -s -o /tmp/qac-ingest.json -w "%{http_code}" \
  -X POST "${WEBHOOK_URL}/api/webhooks/upload?framework=${FRAMEWORK}" \
  -H "X-API-Key: ${API_KEY}" \
  -H "X-Run-Id: ${RUN_ID}" \
  -H "X-Commit-Sha: ${GITHUB_SHA:-}" \
  -H "X-Branch: ${GITHUB_REF_NAME:-}" \
  -H "X-Execution-Env: ${QA_CAPSULE_EXEC_ENV:-STAGING}" \
  -H "X-Execution-Type: ${QA_CAPSULE_EXEC_TYPE:-TEST-RUN}" \
  -F "file=@${FILE}")

echo "QA Capsule HTTP status: ${HTTP_STATUS}"
cat /tmp/qac-ingest.json || true

if [ "$HTTP_STATUS" -ge 400 ]; then
  echo "WARNING: QA Capsule ingestion returned HTTP ${HTTP_STATUS}"
  exit 0
fi

echo "Results queued for processing."
