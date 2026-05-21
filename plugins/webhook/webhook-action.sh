#!/bin/bash
INCIDENT_NAME="${1:-QA Capsule event}"

if [ -z "$WEBHOOK_URL" ]; then
  echo "[ERROR] WEBHOOK_URL is not configured."
  exit 1
fi

PAYLOAD=$(cat <<EOF
{
  "source": "qa-capsule",
  "event": "incident.detected",
  "incident": "${INCIDENT_NAME}",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
)

AUTH_ARGS=()
if [ -n "$WEBHOOK_AUTH_HEADER" ]; then
  AUTH_ARGS=(-H "Authorization: ${WEBHOOK_AUTH_HEADER}")
fi

HTTP=$(curl -s -o /tmp/hook_resp.txt -w "%{http_code}" -X POST \
  -H "Content-Type: application/json" \
  "${AUTH_ARGS[@]}" \
  --data "$PAYLOAD" \
  "$WEBHOOK_URL")

if [ "$HTTP" -ge 200 ] && [ "$HTTP" -lt 300 ]; then
  echo "[WEBHOOK] Delivered (HTTP $HTTP)."
  cat /tmp/hook_resp.txt
  exit 0
fi

echo "[ERROR] Webhook returned HTTP $HTTP"
cat /tmp/hook_resp.txt
exit 1
