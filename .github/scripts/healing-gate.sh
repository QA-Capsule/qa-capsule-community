#!/usr/bin/env bash
# Call the self-healing gate endpoint (skips when secrets are missing).
set -euo pipefail

RUN_ID="${1:-local}"
FRAMEWORK="${2:-Unknown}"

WEBHOOK_URL="${WEBHOOK_URL:-${QA_CAPSULE_URL:-}}"
API_KEY="${API_KEY:-}"

if [ -z "$WEBHOOK_URL" ] || [ -z "$API_KEY" ]; then
  echo "Skipping healing gate — QA Capsule secrets not configured."
  exit 0
fi

echo "Calling Self-Healing Gate..."
curl -s \
  -X POST "${WEBHOOK_URL}/api/healing/gate" \
  -H "X-API-Key: ${API_KEY}" \
  -H "X-Run-Id: ${RUN_ID}" \
  -H "X-Framework: ${FRAMEWORK}" \
  | tee /tmp/healing-report.json || true

echo ""
echo "--- Self-Healing Report ---"
if command -v jq &>/dev/null; then
  jq '.' /tmp/healing-report.json 2>/dev/null || cat /tmp/healing-report.json
else
  cat /tmp/healing-report.json 2>/dev/null || true
fi
