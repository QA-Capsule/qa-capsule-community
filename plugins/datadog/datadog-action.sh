#!/bin/bash
INCIDENT_NAME="${1:-QA Capsule alert}"

if [ -z "$DD_API_KEY" ]; then
  echo "[ERROR] DD_API_KEY is not configured."
  exit 1
fi

SITE="${DD_SITE:-datadoghq.com}"
TITLE="QA Capsule: ${INCIDENT_NAME}"
TEXT="Critical test failure reported by QA Capsule. Check Operations dashboard for logs and flaky score."

PAYLOAD=$(cat <<EOF
{
  "title": "${TITLE}",
  "text": "${TEXT}",
  "alert_type": "error",
  "source_type_name": "qa-capsule",
  "tags": ["qa-capsule","ci-quality"]
}
EOF
)

HTTP=$(curl -s -o /tmp/dd_resp.txt -w "%{http_code}" -X POST \
  -H "DD-API-KEY: ${DD_API_KEY}" \
  -H "Content-Type: application/json" \
  --data "$PAYLOAD" \
  "https://api.${SITE}/api/v1/events")

if [ "$HTTP" = "202" ]; then
  echo "[DATADOG] Event created."
  cat /tmp/dd_resp.txt
  exit 0
fi

echo "[ERROR] Datadog API returned HTTP $HTTP"
cat /tmp/dd_resp.txt
exit 1
