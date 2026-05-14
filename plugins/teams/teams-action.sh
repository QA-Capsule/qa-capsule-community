#!/bin/bash

# ==============================================================================
# MS TEAMS ALERT NOTIFIER
# ==============================================================================
# Variables injected by the Go Control Plane:
# $TEAMS_WEBHOOK_URL, $ALERT_NAME, $ALERT_ERROR, $ALERT_STATUS

if [ -z "$TEAMS_WEBHOOK_URL" ]; then
    echo "[ERROR] TEAMS_WEBHOOK_URL is not configured. Aborting."
    exit 1
fi

echo "Preparing payload for Microsoft Teams..."

# 1. Escape special characters to prevent JSON breakage
SAFE_ERROR=$(echo "$ALERT_ERROR" | sed 's/"/\\"/g' | awk '{printf "%s\\n", $0}')

# 2. Build the "MessageCard" (Microsoft's official format)
PAYLOAD=$(cat <<EOF
{
    "@type": "MessageCard",
    "@context": "http://schema.org/extensions",
    "themeColor": "E81123",
    "summary": "SRE Critical Incident",
    "sections": [{
        "activityTitle": "**SRE INCIDENT: $ALERT_NAME**",
        "activitySubtitle": "Telemetry Status: **$ALERT_STATUS**",
        "text": "**Technical Details:** \n\n\`\`\`text\n$SAFE_ERROR\n\`\`\`",
        "markdown": true
    }],
    "potentialAction": [{
        "@type": "OpenUri",
        "name": "View Control Plane",
        "targets": [{ "os": "default", "uri": "http://localhost:9000" }]
    }]
}
EOF
)

# 3. Send HTTP POST request to Microsoft API
HTTP_RESPONSE=$(curl --write-out "%{http_code}" --silent --output /dev/null -X POST -H "Content-Type: application/json" -d "$PAYLOAD" "$TEAMS_WEBHOOK_URL")

if [ "$HTTP_RESPONSE" -eq 200 ]; then
    echo "[SUCCESS] Critical alert successfully delivered to Microsoft Teams channel."
else
    echo "[ERROR] Teams API rejected the payload (HTTP Code: $HTTP_RESPONSE)."
    exit 1
fi