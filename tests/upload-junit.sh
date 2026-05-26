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
MAX_RETRIES="${QA_CAPSULE_UPLOAD_MAX_RETRIES:-4}"
BACKOFF_SECONDS="${QA_CAPSULE_UPLOAD_BACKOFF_SECONDS:-3}"
ATTEMPT=1

while true; do
  set +e
  HTTP_CODE="$(curl -sS -o /tmp/qa_capsule_upload_response.txt -w "%{http_code}" \
    -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=${FRAMEWORK}" \
    -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
    -H "X-Run-Id: ${RUN_ID}" \
    -H "X-Execution-Env: ${EXEC_ENV}" \
    -H "X-Execution-Type: ${EXEC_TYPE}" \
    -F "file=@${JUNIT_FILE}")"
  CURL_EXIT=$?
  set -e

  if [[ "${CURL_EXIT}" -eq 0 && "${HTTP_CODE}" =~ ^2 ]]; then
    echo "Upload accepted by QA Capsule (HTTP ${HTTP_CODE})."
    break
  fi

  echo "Upload attempt ${ATTEMPT}/${MAX_RETRIES} failed (curl=${CURL_EXIT}, http=${HTTP_CODE})."
  if [[ -f /tmp/qa_capsule_upload_response.txt ]]; then
    echo "Response body:"
    sed -n '1,20p' /tmp/qa_capsule_upload_response.txt || true
  fi

  # Retry only transient network/server responses to keep CI simple and reliable.
  if [[ "${ATTEMPT}" -lt "${MAX_RETRIES}" ]] && [[ "${CURL_EXIT}" -ne 0 || "${HTTP_CODE}" =~ ^(429|500|502|503|504)$ ]]; then
    sleep "${BACKOFF_SECONDS}"
    ATTEMPT=$((ATTEMPT + 1))
    continue
  fi

  echo "ERROR: QA Capsule upload failed after ${ATTEMPT} attempt(s)."
  exit 1
done
