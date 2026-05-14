#!/bin/bash

# $1 corresponds to the action passed by the Go script (e.g., "AUTO_EVENT: Playwright E2E")
INCIDENT_NAME=$1

echo "[SLACK PLUGIN] Triggered for incident: $INCIDENT_NAME"

# 1. Global configuration check
if [ -z "$SLACK_WEBHOOK_URL" ] || [ "$SLACK_WEBHOOK_URL" == "https://hooks.slack.com/services/VOTRE/WEBHOOK/ICI" ]; then
    echo "[ERROR] SLACK_WEBHOOK_URL is not configured in the UI."
    exit 1
fi

# 2. Target Channel determination (Dynamic Routing)
# If the project has a SLACK_CHANNEL configured, use it. Otherwise, default to #general
TARGET_CHANNEL=${SLACK_CHANNEL:-"#general"}

echo "[SLACK PLUGIN] Routing alert to channel: $TARGET_CHANNEL"

# 3. Sending formatted alert to Slack via cURL
curl -s -X POST -H 'Content-type: application/json' \
--data "{
    \"channel\": \"$TARGET_CHANNEL\",
    \"attachments\": [
        {
            \"color\": \"#ff4444\",
            \"title\": \"SRE Alert: ${INCIDENT_NAME}\",
            \"text\": \"A critical error has been detected in your pipeline. Please check the QA Capsule Dashboard for more details and analyze technical debt (Flakiness).\",
            \"footer\": \"QA Capsule Auto-Remediation Engine\"
        }
    ]
}" "$SLACK_WEBHOOK_URL"

echo ""
echo "[SLACK PLUGIN] Alert sent successfully"