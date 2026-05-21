#!/bin/bash
INCIDENT_NAME="${1:-QA Capsule failure}"

if [ -z "$ZEPHYR_API_TOKEN" ]; then
  echo "[ERROR] ZEPHYR_API_TOKEN is not configured."
  exit 1
fi

PROJECT_KEY="${ZEPHYR_PROJECT_KEY:-}"
CASE_KEY="${ZEPHYR_TEST_CASE_KEY:-}"

if [ -z "$PROJECT_KEY" ] || [ -z "$CASE_KEY" ]; then
  echo "[ZEPHYR] Skip: ZEPHYR_PROJECT_KEY and ZEPHYR_TEST_CASE_KEY required."
  exit 0
fi

BASE="${ZEPHYR_API_URL:-https://api.zephyrscale.smartbear.com/v2}"
BASE="${BASE%/}"

PAYLOAD=$(cat <<EOF
{
  "projectKey": "${PROJECT_KEY}",
  "testCaseKey": "${CASE_KEY}",
  "statusName": "Fail",
  "comment": "QA Capsule: ${INCIDENT_NAME}"
}
EOF
)

HTTP=$(curl -s -o /tmp/zephyr_resp.txt -w "%{http_code}" -X POST \
  -H "Authorization: Bearer ${ZEPHYR_API_TOKEN}" \
  -H "Content-Type: application/json" \
  --data "$PAYLOAD" \
  "${BASE}/testexecutions")

if [ "$HTTP" = "201" ] || [ "$HTTP" = "200" ]; then
  echo "[ZEPHYR] Test execution created."
  cat /tmp/zephyr_resp.txt
  exit 0
fi

echo "[ERROR] Zephyr Scale API returned HTTP $HTTP"
cat /tmp/zephyr_resp.txt
exit 1
