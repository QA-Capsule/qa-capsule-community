#!/bin/bash
INCIDENT_NAME="${1:-flaky test}"

if [ -z "$QA_WEBHOOK_URL" ]; then
  echo "[SKIP] QA_WEBHOOK_URL not set — configure in Plugin Engine to enable flaky summaries."
  exit 0
fi

PAYLOAD=$(cat <<EOF
{
  "source": "qa-capsule",
  "type": "flaky_summary",
  "test": "${INCIDENT_NAME}",
  "message": "Flaky or unstable test detected. Consider quarantine, deflake sprint, or ownership assignment.",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
)

HTTP=$(curl -s -o /tmp/qa_resp.txt -w "%{http_code}" -X POST \
  -H "Content-Type: application/json" \
  --data "$PAYLOAD" \
  "$QA_WEBHOOK_URL")

if [ "$HTTP" -ge 200 ] && [ "$HTTP" -lt 300 ]; then
  echo "[QA] Flaky summary sent (HTTP $HTTP)."
  exit 0
fi

echo "[ERROR] QA webhook returned HTTP $HTTP"
cat /tmp/qa_resp.txt
exit 1
