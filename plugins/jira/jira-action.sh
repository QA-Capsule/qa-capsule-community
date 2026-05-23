#!/bin/bash
# $1 = incident name or MANUAL (Plugin Engine test)
INCIDENT_NAME="${1:-QA Capsule incident}"
IS_MANUAL=false
if [ "$INCIDENT_NAME" = "MANUAL" ]; then
  IS_MANUAL=true
  INCIDENT_NAME="[QA Capsule] Manual test from Plugin Engine"
fi

echo "[JIRA PLUGIN] Attempting to create ticket for: $INCIDENT_NAME"

if [ -z "$JIRA_URL" ] || [ -z "$JIRA_EMAIL" ] || [ -z "$JIRA_API_TOKEN" ] || [ "$JIRA_URL" = "https://YOUR-DOMAIN.atlassian.net" ]; then
  echo "[ERROR] Jira API configuration missing. Set JIRA_URL, JIRA_EMAIL, and JIRA_API_TOKEN in Plugin Engine → Configure."
  exit 1
fi

# Project key: CI/CD Gateway (auto) overrides plugin default (manual + fallback)
PROJECT_KEY="${JIRA_PROJECT_KEY:-}"
if [ -z "$PROJECT_KEY" ]; then
  if [ "$IS_MANUAL" = true ]; then
    echo "[ERROR] JIRA_PROJECT_KEY is required for manual Execute."
    echo "[HINT] Plugin Engine → Jira → Configure → JIRA_PROJECT_KEY (e.g. PROJ, QA, DEV)."
    echo "[HINT] Or set Jira project key on your CI/CD Gateway for auto-trigger only."
    exit 1
  fi
  echo "[JIRA PLUGIN] Skipped: no JIRA_PROJECT_KEY on this pipeline (CI/CD Gateway → Edit → Jira project key)."
  exit 0
fi

ISSUE_TYPE="${JIRA_ISSUE_TYPE:-Bug}"
JIRA_BASE="${JIRA_URL%/}"

echo "[JIRA PLUGIN] Project: ${PROJECT_KEY} · Issue type: ${ISSUE_TYPE}"

# Escape summary for JSON (minimal)
SAFE_SUMMARY=$(echo "$INCIDENT_NAME" | sed 's/\\/\\\\/g; s/"/\\"/g')
SAFE_DESC="Incident reported by QA Capsule.\\n\\nSummary: ${SAFE_SUMMARY}\\n\\nOpen the Operations dashboard for logs and flaky classification."

PAYLOAD=$(cat <<EOF
{
  "fields": {
    "project": { "key": "${PROJECT_KEY}" },
    "summary": "${SAFE_SUMMARY}",
    "description": "${SAFE_DESC}",
    "issuetype": { "name": "${ISSUE_TYPE}" }
  }
}
EOF
)

RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X POST \
  -H "Content-Type: application/json" \
  -u "${JIRA_EMAIL}:${JIRA_API_TOKEN}" \
  --data "$PAYLOAD" \
  "${JIRA_BASE}/rest/api/2/issue")

HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS:" | awk -F: '{print $2}')
BODY=$(echo "$RESPONSE" | sed -e 's/HTTP_STATUS:.*//g')

if [ "$HTTP_STATUS" = "201" ]; then
  ISSUE_KEY=$(echo "$BODY" | grep -o '"key":"[^"]*' | head -1 | sed 's/"key":"//')
  echo "[JIRA PLUGIN] Success! Ticket created: ${ISSUE_KEY}"
  echo "[JIRA PLUGIN] Browse: ${JIRA_BASE}/browse/${ISSUE_KEY}"
  exit 0
fi

echo "[ERROR] Failed to create Jira ticket (HTTP ${HTTP_STATUS})."
echo "[DETAIL] $BODY"
echo "[HINT] Check JIRA_PROJECT_KEY exists, issue type \"${ISSUE_TYPE}\" is valid for this project, and the API token has create issues permission."
exit 1
