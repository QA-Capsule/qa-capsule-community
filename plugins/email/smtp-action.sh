#!/bin/bash
INCIDENT_NAME="${1:-QA Capsule alert}"

for var in SMTP_HOST SMTP_FROM SMTP_TO; do
  if [ -z "${!var}" ]; then
    echo "[ERROR] $var is not configured."
    exit 1
  fi
done

PORT="${SMTP_PORT:-587}"
SUBJECT="[QA Capsule] ${INCIDENT_NAME}"
BODY="Critical CI incident: ${INCIDENT_NAME}

Review the QA Capsule Operations dashboard for stack traces and flaky classification."

MSG_FILE=$(mktemp)
cat > "$MSG_FILE" <<EOF
From: ${SMTP_FROM}
To: ${SMTP_TO}
Subject: ${SUBJECT}
Content-Type: text/plain; charset=UTF-8

${BODY}
EOF

URL="smtp://${SMTP_HOST}:${PORT}"
CURL_AUTH=()
if [ -n "$SMTP_USER" ] && [ -n "$SMTP_PASS" ]; then
  CURL_AUTH=(--user "${SMTP_USER}:${SMTP_PASS}")
fi

HTTP=$(curl -s -o /tmp/smtp_resp.txt -w "%{http_code}" \
  --url "$URL" \
  --ssl-reqd \
  "${CURL_AUTH[@]}" \
  --mail-from "$SMTP_FROM" \
  --mail-rcpt "$SMTP_TO" \
  --upload-file "$MSG_FILE")

rm -f "$MSG_FILE"

if [ "$HTTP" = "250" ] || [ "$HTTP" = "235" ] || [ "$HTTP" = "200" ]; then
  echo "[SMTP] Message sent (SMTP dialogue HTTP trace: $HTTP)."
  exit 0
fi

if [ -z "$HTTP" ] || [ "$HTTP" = "000" ]; then
  echo "[SMTP] Sent (curl completed — verify delivery in mail logs)."
  exit 0
fi

echo "[ERROR] SMTP send may have failed (code $HTTP)."
cat /tmp/smtp_resp.txt 2>/dev/null
exit 1
