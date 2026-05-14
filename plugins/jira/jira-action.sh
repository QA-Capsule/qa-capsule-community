#!/bin/bash

# $1 corresponds to the action or incident name passed by the Go engine
INCIDENT_NAME=$1

echo "[JIRA PLUGIN] Attempting to create ticket for: $INCIDENT_NAME"

# 1. Check plugin global variables
if [ -z "$JIRA_URL" ] || [ -z "$JIRA_EMAIL" ] || [ -z "$JIRA_API_TOKEN" ] || [ "$JIRA_URL" == "https://YOUR-DOMAIN.atlassian.net" ]; then
    echo "[ERROR] Jira API configuration (URL, Email, or Token) is missing in the UI."
    exit 1
fi

# 2. Check project-specific routing
# If the project has no JIRA_PROJECT_KEY defined, abort silently
if [ -z "$JIRA_PROJECT_KEY" ] || [ "$JIRA_PROJECT_KEY" == "" ]; then
    echo "[JIRA PLUGIN] Aborting: No JIRA_PROJECT_KEY configured for this pipeline."
    exit 0
fi

echo "[JIRA PLUGIN] Routing to Jira project: $JIRA_PROJECT_KEY"

# 3. Prepare the JSON payload for Jira REST API
PAYLOAD=$(cat <<EOF
{
    "fields": {
        "project": {
            "key": "$JIRA_PROJECT_KEY"
        },
        "summary": "[QA Capsule] $INCIDENT_NAME",
        "description": "A critical error was detected by the QA Capsule Control Plane.\n\nIncident: $INCIDENT_NAME\nRequired action: Check the logs and associated technical debt in the Dashboard.",
        "issuetype": {
            "name": "Bug"
        }
    }
}
EOF
)

# 4. Send request to Jira API
RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X POST -H 'Content-Type: application/json' \
    -u "${JIRA_EMAIL}:${JIRA_API_TOKEN}" \
    --data "$PAYLOAD" \
    "${JIRA_URL}/rest/api/2/issue")

# Extract HTTP status code
HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS:" | awk -F: '{print $2}')
BODY=$(echo "$RESPONSE" | sed -e 's/HTTP_STATUS:.*//g')

if [ "$HTTP_STATUS" -eq 201 ]; then
    ISSUE_KEY=$(echo "$BODY" | grep -o '"key":"[^"]*' | grep -o '[^"]*$')
    echo "[JIRA PLUGIN] Success! Ticket created with key: $ISSUE_KEY"
else
    echo "[ERROR] Failed to create Jira ticket (HTTP $HTTP_STATUS)."
    echo "[DETAIL] $BODY"
    exit 1
fi