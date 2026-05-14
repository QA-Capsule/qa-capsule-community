---
icon: fontawesome/brands/jira
---

# Jira Software Integration

The Jira plugin is one of the most powerful automated remediation tools in QA Capsule. Instead of having QA engineers manually copy-paste terminal logs into bug reports, QA Capsule can automatically generate comprehensive Jira tickets the second a critical test fails.

These tickets include the exact failing test name, the environment, and a neatly formatted, code-blocked StackTrace, instantly assigning it to the correct project board.

This guide covers both the **Jira-side configuration** and the **QA Capsule-side setup**.

---

## Part 1: Configuration on the Jira Side

To allow QA Capsule to create tickets on your behalf, it must authenticate with Jira's REST API. Atlassian no longer allows basic password authentication; you must generate a dedicated **API Token**.

### Step 1: Create a Service Account (Recommended)
While you can use your personal Jira account, it is considered a best practice to create a dedicated "Service Account" (e.g., `sre-bot@yourcompany.com`).
1. Ask your Jira Administrator to invite the Service Account email to your Atlassian workspace.
2. Ensure this account has **Create Issues** permissions in the projects you want to monitor.

### Step 2: Generate an Atlassian API Token
You must generate the token while logged in as the account that will create the tickets.

1. Log in to Jira Software.
2. Click on your profile avatar in the top right corner and select **Manage your account**.
3. Navigate to the **Security** tab in the top menu.
4. Under the **API token** section, click on **Create and manage API tokens**.
   *(Placeholder for your image: `![Jira API Token Menu](../assets/jira-token-menu.png)`)*
5. Click the **Create API token** button.
6. **Label:** Name it something recognizable, like `QA_Capsule_Bot`.
7. Click **Create**.
8. **CRITICAL:** Click the **Copy** button immediately. Atlassian will *never* show you this token again. Store it temporarily in a secure notepad.

### Step 3: Identify your Jira Project Keys
Jira categorizes tickets into Projects, and each Project has a unique "Key" (usually 2 to 4 uppercase letters).
1. Open your Jira Software board.
2. Look at any existing ticket on the board (e.g., `PAY-1042` or `WEB-89`).
3. The letters before the hyphen (`PAY`, `WEB`) represent your **Project Key**. You will need this to route alerts dynamically.

---

## Part 2: QA Capsule Global Configuration

Now that Jira is ready to accept requests, we need to configure the global variables inside the QA Capsule Plugin Engine.

1. Open your **QA Capsule Dashboard**.
2. Navigate to the **Plugin Engine** using the left-hand sidebar.
3. Locate the **Jira Ticket Creator** plugin card.
4. Click the **Configure** button.
5. Fill in the global environment variables:

   * `JIRA_URL`: Your base Atlassian URL (e.g., `https://mycompany.atlassian.net`). *Do not add a trailing slash.*
   * `JIRA_USER`: The exact email address of the account you used to generate the token (e.g., `sre-bot@mycompany.com`).
   * `JIRA_API_TOKEN`: Paste the token you copied in Step 2.
   
6. Click **Save Configuration**.

---

## Part 3: Project-Level Dynamic Routing

Because QA Capsule is multi-tenant, it needs to know *which* Jira board to send the ticket to when a specific pipeline fails.

1. In QA Capsule, go to the **CI/CD Gateways** module.
2. When you provision a new endpoint (or edit an existing one), locate the **Routing: Jira Project Key** field.
3. Enter the exact Jira Project Key you identified in Step 1 (e.g., `PAY`).
4. Click **Provision Project Endpoint**.

When the "Payment API" pipeline fails, the Plugin Engine will dynamically inject `JIRA_PROJECT_KEY="PAY"` into the plugin script!

---

## Part 4: The Plugin Script Implementation (`jira.sh`)

If you are a system administrator looking to understand or modify how QA Capsule talks to Jira, here is the underlying Bash script used by the plugin. 

It utilizes the Jira REST API v2 and `jq` to safely format the multi-line JSON payload.

### The `manifest.json`

```json
{
  "name": "Jira Bug Creator",
  "description": "Automatically creates Bug tickets in Jira with full stacktraces.",
  "version": "1.0.0",
  "author": "SRE Team",
  "entrypoint": "jira.sh",
  "global_env": [
    "JIRA_URL",
    "JIRA_USER",
    "JIRA_API_TOKEN"
  ],
  "trigger_on": [
    "[FATAL]",
    "CRITICAL"
  ]
}
```

### The `jira.sh` Executable

```Bash

#!/bin/bash
# ==============================================================================
# QA Capsule Plugin: Jira Ticket Creator
# ==============================================================================

# 1. Validate Global Variables
if [ -z "$JIRA_URL" ] || [ -z "$JIRA_USER" ] || [ -z "$JIRA_API_TOKEN" ]; then
  echo "[ERROR] Jira global authentication variables are missing."
  exit 1
fi

# 2. Validate Dynamic Routing Context
if [ -z "$JIRA_PROJECT_KEY" ]; then
  echo "[SKIP] No JIRA_PROJECT_KEY defined for this project. Aborting."
  exit 0
fi

echo "[INFO] Preparing to create Jira Bug in project: $JIRA_PROJECT_KEY"

# 3. Format the Description safely using jq
# We wrap the stacktrace in Jira's markdown code blocks {code:java}...{code}
FORMATTED_DESC="*Pipeline Failure Detected by QA Capsule*\n\n*Error Summary:* $INCIDENT_ERROR\n*Environment Context:* $INCIDENT_BROWSER\n\n*Console Logs & StackTrace:*\n{code:java}\n$INCIDENT_LOGS\n{code}"

SAFE_DESC=$(echo "$FORMATTED_DESC" | jq -R -s '.')

# 4. Construct the Jira v2 REST API JSON Payload
JSON_PAYLOAD=$(cat <<EOF
{
  "fields": {
    "project": {
      "key": "$JIRA_PROJECT_KEY"
    },
    "summary": "[CI/CD] $INCIDENT_NAME",
    "description": ${SAFE_DESC},
    "issuetype": {
      "name": "Bug"
    }
  }
}
EOF
)

# 5. Execute the HTTP Request via cURL
# We use basic auth with the email and API Token
API_ENDPOINT="$JIRA_URL/rest/api/2/issue"

HTTP_RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X POST "$API_ENDPOINT" \
  --user "$JIRA_USER:$JIRA_API_TOKEN" \
  -H "Accept: application/json" \
  -H "Content-Type: application/json" \
  -d "$JSON_PAYLOAD")

# 6. Parse Response
HTTP_BODY=$(echo "$HTTP_RESPONSE" | sed -e '$ d')
HTTP_STATUS=$(echo "$HTTP_RESPONSE" | tail -n1 | sed -e 's/HTTP_STATUS://')

if [ "$HTTP_STATUS" -eq 201 ]; then
  TICKET_KEY=$(echo "$HTTP_BODY" | jq -r '.key')
  echo "[SUCCESS] Jira ticket created successfully: $TICKET_KEY"
  exit 0
else
  echo "[ERROR] Failed to create Jira ticket. HTTP $HTTP_STATUS"
  echo "Response from Jira: $HTTP_BODY"
  exit 1
fi
```

## Part 5: Troubleshooting

If your Jira tickets are not being created, check the Stdout Logs terminal in the QA Capsule Plugin UI. Here are common Jira API errors:

1. **HTTP 401 Unauthorized :** Your `JIRA_API_TOKEN` or `JIRA_USER` is incorrect. Ensure the user email exactly matches the account that generated the token.
2. **HTTP 400 Bad Request (Issue Type not found)** : The script attempts to create an issue of type "Bug". Some heavily customized Jira instances rename "Bug" to something else (like "Defect"). If this happens, an Admin must edit the `jira.sh` script to match your custom Issue Type name.
3. **HTTP 400 Bad Request (Field 'reporter' is required) :** Some Jira boards have strict custom required fields. If your board requires fields other than Summary and Description, you will need to add those fields to the `JSON_PAYLOAD` in the script.