#!/bin/bash
INCIDENT_NAME="${1:-QA Capsule alert}"

if [ -z "$PAGERDUTY_ROUTING_KEY" ]; then
  echo "[ERROR] PAGERDUTY_ROUTING_KEY is not configured in the Plugin Engine."
  exit 1
fi

SUMMARY="[QA Capsule] ${INCIDENT_NAME}"
DETAIL="Automated alert from QA Capsule. Review the Operations dashboard for stack traces and flaky classification."

PAYLOAD=$(cat <<EOF
{
  "routing_key": "${PAGERDUTY_ROUTING_KEY}",
  "event_action": "trigger",
  "payload": {
    "summary": "${SUMMARY}",
    "source": "qa-capsule",
    "severity": "critical",
    "custom_details": {
      "incident": "${INCIDENT_NAME}",
      "detail": "${DETAIL}"
    }
  }
}
EOF
)

HTTP=$(curl -s -o /tmp/pd_resp.txt -w "%{http_code}" -X POST \
  -H "Content-Type: application/json" \
  --data "$PAYLOAD" \
  "https://events.pagerduty.com/v2/enqueue")

if [ "$HTTP" = "202" ]; then
  echo "[PAGERDUTY] Incident queued successfully."
  cat /tmp/pd_resp.txt
  exit 0
fi

echo "[ERROR] PagerDuty API returned HTTP $HTTP"
cat /tmp/pd_resp.txt
exit 1
