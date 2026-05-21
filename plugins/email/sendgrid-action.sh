#!/bin/bash
INCIDENT_NAME="${1:-QA Capsule alert}"

for var in SENDGRID_API_KEY SENDGRID_FROM SENDGRID_TO; do
  if [ -z "${!var}" ]; then
    echo "[ERROR] $var is not configured."
    exit 1
  fi
done

SUBJECT="[QA Capsule] ${INCIDENT_NAME}"
BODY="A critical CI quality incident was detected. Open the Operations dashboard for details, logs, and flaky analysis."

PAYLOAD=$(cat <<EOF
{
  "personalizations": [{"to": [{"email": "${SENDGRID_TO}"}]}],
  "from": {"email": "${SENDGRID_FROM}"},
  "subject": "${SUBJECT}",
  "content": [{"type": "text/html", "value": "<p>${BODY}</p><p><strong>Incident:</strong> ${INCIDENT_NAME}</p>"}]
}
EOF
)

HTTP=$(curl -s -o /tmp/sg_resp.txt -w "%{http_code}" -X POST \
  -H "Authorization: Bearer ${SENDGRID_API_KEY}" \
  -H "Content-Type: application/json" \
  --data "$PAYLOAD" \
  "https://api.sendgrid.com/v3/mail/send")

if [ "$HTTP" = "202" ] || [ "$HTTP" = "200" ]; then
  echo "[SENDGRID] Email accepted (HTTP $HTTP)."
  exit 0
fi

echo "[ERROR] SendGrid returned HTTP $HTTP"
cat /tmp/sg_resp.txt
exit 1
