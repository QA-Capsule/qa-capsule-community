#!/bin/bash
INCIDENT_NAME="${1:-QA Capsule failure}"

for var in XRAY_CLIENT_ID XRAY_CLIENT_SECRET; do
  if [ -z "${!var}" ]; then
    echo "[ERROR] $var is not configured."
    exit 1
  fi
done

TEST_KEY="${XRAY_TEST_KEY:-}"
PROJECT="${XRAY_PROJECT_KEY:-JIRA}"

if [ -z "$TEST_KEY" ]; then
  echo "[XRAY] Skip: XRAY_TEST_KEY required (e.g. PROJ-123 test issue key)."
  exit 0
fi

AUTH_PAYLOAD=$(cat <<EOF
{"client_id": "${XRAY_CLIENT_ID}", "client_secret": "${XRAY_CLIENT_SECRET}"}
EOF
)

TOKEN=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  --data "$AUTH_PAYLOAD" \
  "https://xray.cloud.getxray.app/api/v2/authenticate" | tr -d '"')

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
  echo "[ERROR] Xray authentication failed."
  exit 1
fi

# Minimal JUnit-style import for a single failed test
IMPORT_PAYLOAD=$(cat <<EOF
{
  "testExecutionKey": "${TEST_KEY}",
  "info": {
    "summary": "QA Capsule execution",
    "project": "${PROJECT}"
  },
  "tests": [
    {
      "testKey": "${TEST_KEY}",
      "status": "FAILED",
      "comment": "QA Capsule: ${INCIDENT_NAME}"
    }
  ]
}
EOF
)

HTTP=$(curl -s -o /tmp/xray_resp.txt -w "%{http_code}" -X POST \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  --data "$IMPORT_PAYLOAD" \
  "https://xray.cloud.getxray.app/api/v2/import/execution")

if [ "$HTTP" = "200" ] || [ "$HTTP" = "201" ]; then
  echo "[XRAY] Execution imported."
  cat /tmp/xray_resp.txt
  exit 0
fi

echo "[ERROR] Xray Cloud API returned HTTP $HTTP"
cat /tmp/xray_resp.txt
exit 1
