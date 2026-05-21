#!/bin/bash
INCIDENT_NAME="${1:-QA Capsule alert}"

if [ -z "$VICTOROPS_ROUTING_URL" ]; then
  echo "[ERROR] VICTOROPS_ROUTING_URL is not configured."
  echo "[HINT] Use the full REST integration URL from Splunk On-Call (VictorOps) routing keys."
  exit 1
fi

ENTITY_ID="qa-capsule-$(date +%s)"
PAYLOAD=$(cat <<EOF
{
  "message_type": "CRITICAL",
  "entity_id": "${ENTITY_ID}",
  "entity_display_name": "QA Capsule",
  "state_message": "[QA Capsule] ${INCIDENT_NAME}",
  "alias": "${ENTITY_ID}"
}
EOF
)

HTTP=$(curl -s -o /tmp/vo_resp.txt -w "%{http_code}" -X POST \
  -H "Content-Type: application/json" \
  --data "$PAYLOAD" \
  "$VICTOROPS_ROUTING_URL")

if [ "$HTTP" -ge 200 ] && [ "$HTTP" -lt 300 ]; then
  echo "[VICTOROPS] Alert posted (HTTP $HTTP)."
  cat /tmp/vo_resp.txt
  exit 0
fi

echo "[ERROR] VictorOps / Splunk On-Call returned HTTP $HTTP"
cat /tmp/vo_resp.txt
exit 1
