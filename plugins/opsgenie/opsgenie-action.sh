#!/bin/bash
INCIDENT_NAME="${1:-QA Capsule alert}"

if [ -z "$OPSGENIE_API_KEY" ]; then
  echo "[ERROR] OPSGENIE_API_KEY is not configured."
  exit 1
fi

BASE="${OPSGENIE_API_URL:-https://api.opsgenie.com}"
BASE="${BASE%/}"

PAYLOAD=$(cat <<EOF
{
  "message": "[QA Capsule] ${INCIDENT_NAME}",
  "description": "Critical failure detected by QA Capsule. Review Operations dashboard for logs and flaky classification.",
  "priority": "P1",
  "source": "qa-capsule",
  "tags": ["qa-capsule", "ci-quality"]
}
EOF
)

HTTP=$(curl -s -o /tmp/og_resp.txt -w "%{http_code}" -X POST \
  -H "Authorization: GenieKey ${OPSGENIE_API_KEY}" \
  -H "Content-Type: application/json" \
  --data "$PAYLOAD" \
  "${BASE}/v2/alerts")

if [ "$HTTP" = "202" ] || [ "$HTTP" = "200" ]; then
  echo "[OPSGENIE] Alert created (HTTP $HTTP)."
  cat /tmp/og_resp.txt
  exit 0
fi

echo "[ERROR] Opsgenie API returned HTTP $HTTP"
cat /tmp/og_resp.txt
exit 1
