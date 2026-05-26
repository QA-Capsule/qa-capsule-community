#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "Usage: tests/upload-junit.sh <framework> <junit_file>"
  exit 1
fi

FRAMEWORK="$1"
JUNIT_FILE="$2"

QA_CAPSULE_URL="${QA_CAPSULE_URL:-${WEBHOOK_URL:-}}"
QA_CAPSULE_API_KEY="${QA_CAPSULE_API_KEY:-${API_KEY:-}}"
RUN_ID="${CI_PIPELINE_ID:-${GITHUB_RUN_ID:-local-$(date +%s)}}"
EXEC_ENV="${QA_CAPSULE_EXEC_ENV:-DEV}"
EXEC_TYPE="${QA_CAPSULE_EXEC_TYPE:-TEST-RUN}"

if [[ ! -f "${JUNIT_FILE}" ]]; then
  echo "ERROR: JUnit file not found: ${JUNIT_FILE}"
  exit 1
fi

if [[ -z "${QA_CAPSULE_URL}" || -z "${QA_CAPSULE_API_KEY}" ]]; then
  echo "INFO: QA_CAPSULE_URL / QA_CAPSULE_API_KEY not set; upload skipped."
  exit 0
fi

echo "Uploading ${FRAMEWORK} JUnit file: ${JUNIT_FILE}"
curl -f -S -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=${FRAMEWORK}" \
  -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
  -H "X-Run-Id: ${RUN_ID}" \
  -H "X-Execution-Env: ${EXEC_ENV}" \
  -H "X-Execution-Type: ${EXEC_TYPE}" \
  -F "file=@${JUNIT_FILE}"
