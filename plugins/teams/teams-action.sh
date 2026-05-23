#!/bin/bash
# $1 = incident name or MANUAL · TEAMS_WEBHOOK_URL from plugin Configure or CI/CD Gateway

ALERT_NAME="${ALERT_NAME:-${INCIDENT_NAME:-$1}}"
ALERT_STATUS="${ALERT_STATUS:-CRITICAL}"
ALERT_ERROR="${ALERT_ERROR:-QA Capsule alert — open the Operations dashboard for details.}"

if [ -z "$TEAMS_WEBHOOK_URL" ]; then
  echo "[ERROR] TEAMS_WEBHOOK_URL is not configured."
  echo "[HINT] Plugin Engine → Configure, or CI/CD Gateway → Teams webhook URL for auto-trigger."
  exit 1
fi

echo "[TEAMS] Sending MessageCard for: $ALERT_NAME"

SAFE_ERROR=$(echo "$ALERT_ERROR" | sed 's/"/\\"/g' | awk '{printf "%s\\n", $0}')

PAYLOAD=$(cat <<EOF
{
  "@type": "MessageCard",
  "@context": "http://schema.org/extensions",
  "themeColor": "E81123",
  "summary": "QA Capsule alert",
  "sections": [{
    "activityTitle": "SRE incident: ${ALERT_NAME}",
    "activitySubtitle": "Status: ${ALERT_STATUS}",
    "text": "**Details:**\n\n\`\`\`text\n${SAFE_ERROR}\n\`\`\`",
    "markdown": true
  }]
}
EOF
)

HTTP_RESPONSE=$(curl --write-out "%{http_code}" --silent --output /dev/null \
  -X POST -H "Content-Type: application/json" -d "$PAYLOAD" "$TEAMS_WEBHOOK_URL")

if [ "$HTTP_RESPONSE" = "200" ] || [ "$HTTP_RESPONSE" = "202" ]; then
  echo "[TEAMS] Success (HTTP $HTTP_RESPONSE)."
  exit 0
fi

echo "[ERROR] Teams API rejected the payload (HTTP $HTTP_RESPONSE)."
exit 1
