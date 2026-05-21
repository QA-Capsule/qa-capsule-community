#!/bin/bash
INCIDENT_NAME="${1:-QA Capsule failure}"

for var in TESTRAIL_URL TESTRAIL_USER TESTRAIL_API_KEY; do
  if [ -z "${!var}" ]; then
    echo "[ERROR] $var is not configured."
    exit 1
  fi
done

RUN_ID="${TESTRAIL_RUN_ID:-}"
CASE_ID="${TESTRAIL_CASE_ID:-}"

if [ -z "$RUN_ID" ] || [ -z "$CASE_ID" ]; then
  echo "[TESTRAIL] Skip: TESTRAIL_RUN_ID and TESTRAIL_CASE_ID required (plugin config or project routing)."
  exit 0
fi

BASE="${TESTRAIL_URL%/}"
COMMENT="Failed via QA Capsule: ${INCIDENT_NAME}"

PAYLOAD=$(cat <<EOF
{
  "status_id": 5,
  "comment": "${COMMENT}"
}
EOF
)

URL="${BASE}/index.php?/api/v2/add_result/${RUN_ID}/${CASE_ID}"
HTTP=$(curl -s -o /tmp/tr_resp.txt -w "%{http_code}" -X POST \
  -u "${TESTRAIL_USER}:${TESTRAIL_API_KEY}" \
  -H "Content-Type: application/json" \
  --data "$PAYLOAD" \
  "$URL")

if [ "$HTTP" = "200" ]; then
  echo "[TESTRAIL] Result recorded for case ${CASE_ID} on run ${RUN_ID}."
  cat /tmp/tr_resp.txt
  exit 0
fi

echo "[ERROR] TestRail API returned HTTP $HTTP"
cat /tmp/tr_resp.txt
exit 1
