---
icon: material/microsoft-teams
---

# Microsoft Teams Integration

The Microsoft Teams plugin allows your SRE and QA teams to receive rich, actionable incident notifications directly within their Teams channels. Unlike standard text notifications, QA Capsule leverages **Microsoft Adaptive Cards** to provide a structured layout, including error summaries, environment metadata, and formatted stacktraces.

With **Dynamic Routing**, you can configure different Teams channels for different projects (e.g., `#Platform-Alerts` for backend services and `#UI-Bugs` for frontend apps) using a single plugin configuration.

---

## Part 1: Microsoft Teams Configuration

To enable QA Capsule to post messages to a Teams channel, you must create an **Incoming Webhook** connector for each destination channel.

### Step 1: Create an Incoming Webhook in Teams
1. Open **Microsoft Teams** and navigate to the specific Team and Channel where you want to receive alerts.
2. Click on the **More options** (`...`) next to the channel name and select **Connectors**.
3. Search for **Incoming Webhook** in the list of available connectors.
4. Click **Add** (or **Configure** if already added).
5. Provide a name for your webhook (e.g., `QA Capsule Bot`).
6. (Optional) Upload the QA Capsule logo as the profile picture for the bot.
7. Click **Create**.
8. **CRITICAL:** Microsoft will generate a unique URL. **Copy this URL immediately**. It will look like: `https://yourcompany.webhook.office.com/webhookb2/uuid/IncomingWebhook/id/uuid`.
9. Click **Done**.

---

## Part 2: QA Capsule Global Configuration

Unlike the Slack plugin (which often uses a single global token), Microsoft Teams webhooks are strictly bound to a specific channel. Therefore, the primary configuration happens at the **Project Level**.

However, you still need to ensure the plugin is enabled globally:

1. Open your **QA Capsule Dashboard**.
2. Navigate to the **Plugin Engine** tab.
3. Locate the **MS Teams Notifier** plugin card.
4. Ensure the status switch is toggled to **AUTO-RUN ON**.
5. Click **Configure** to verify if any global environment variables (like a corporate proxy URL) are required for your specific network setup.

---

## Part 3: Project-Level Dynamic Routing

This is where you link your Microsoft Teams channel to a specific CI/CD pipeline.

1. Navigate to the **CI/CD Gateways** module.
2. Select the project you wish to configure (or provision a new one).
3. Locate the **Routing: MS Teams Webhook** input field.
4. Paste the **Webhook URL** you copied from Teams in Part 1.
5. Click **Provision Project Endpoint**.

Whenever this project triggers a `CRITICAL` or `[FATAL]` alert, the QA Capsule engine will retrieve this specific URL from the database and inject it into the plugin execution environment as the variable `TEAMS_WEBHOOK`.

---

## Part 4: The Plugin Script Implementation (`teams.sh`)

For developers who want to customize the visual layout of the Teams card, here is the underlying implementation. We use the **Adaptive Card JSON schema** to ensure the logs are readable and the alert stands out.

### The `manifest.json`
```json
{
  "name": "MS Teams Notifier",
  "description": "Sends rich Adaptive Cards to Microsoft Teams channels.",
  "version": "1.0.0",
  "author": "SRE Team",
  "entrypoint": "teams.sh",
  "global_env": [],
  "trigger_on": [
    "[FATAL]",
    "CRITICAL"
  ]
}
```

### The `teams.sh` Executable

```Bash
#!/bin/bash
# ==============================================================================
# QA Capsule Plugin: Microsoft Teams Notifier (Adaptive Cards)
# ==============================================================================

# 1. Validate Dynamic Routing Context
if [ -z "$TEAMS_WEBHOOK" ]; then
  echo "[SKIP] No TEAMS_WEBHOOK defined for this project. Aborting."
  exit 0
fi

echo "[INFO] Formatting Adaptive Card for Microsoft Teams..."

# 2. Safely escape multi-line logs using jq
# Teams requires strict JSON escaping for the Adaptive Card 'text' blocks
SAFE_NAME=$(echo "$INCIDENT_NAME" | jq -R -s '.')
SAFE_ERROR=$(echo "$INCIDENT_ERROR" | jq -R -s '.')
SAFE_LOGS=$(echo "$INCIDENT_LOGS" | jq -R -s '.')

# 3. Construct the Adaptive Card JSON Payload
# We use a 'FactSet' for metadata and a code-block styled text for logs
JSON_PAYLOAD=$(cat <<EOF
{
    "type": "message",
    "attachments": [
        {
            "contentType": "application/vnd.microsoft.card.adaptive",
            "content": {
                "type": "AdaptiveCard",
                "body": [
                    {
                        "type": "TextBlock",
                        "size": "Medium",
                        "weight": "Bolder",
                        "text": "QA Capsule Alert: $INCIDENT_NAME",
                        "color": "Attention"
                    },
                    {
                        "type": "FactSet",
                        "facts": [
                            { "title": "Status:", "value": "$INCIDENT_STATUS" },
                            { "title": "Environment:", "value": "$INCIDENT_BROWSER" },
                            { "title": "Summary:", "value": ${SAFE_ERROR} }
                        ]
                    },
                    {
                        "type": "TextBlock",
                        "text": "Stacktrace & Console Logs:",
                        "wrap": true,
                        "weight": "Bolder"
                    },
                    {
                        "type": "Container",
                        "style": "emphasis",
                        "items": [
                            {
                                "type": "TextBlock",
                                "text": ${SAFE_LOGS},
                                "wrap": true,
                                "fontType": "Monospace",
                                "size": "Small"
                            }
                        ]
                    }
                ],
                "\$schema": "[http://adaptivecards.io/schemas/adaptive-card.json](http://adaptivecards.io/schemas/adaptive-card.json)",
                "version": "1.4"
            }
        }
    ]
}
EOF
)

# 4. Dispatch the HTTP Request to Microsoft Teams
echo "[INFO] Pushing Adaptive Card to Teams..."

HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
  -H "Content-Type: application/json" \
  -d "$JSON_PAYLOAD" \
  "$TEAMS_WEBHOOK")

# 5. Handle Response
if [ "$HTTP_STATUS" -eq 200 ] || [ "$HTTP_STATUS" -eq 201 ]; then
  echo "[SUCCESS] Message successfully delivered to MS Teams."
  exit 0
else
  echo "[ERROR] MS Teams API returned error code: $HTTP_STATUS"
  exit 1
fi
```

## Part 5: Troubleshooting

Microsoft Teams can be very strict about the JSON structure of Adaptive Cards. If your notifications are not appearing:

1. **HTTP 400 Bad Request:** This is the most common error. It means the `JSON_PAYLOAD` is invalid. This usually happens if your INCIDENT_LOGS contains special characters (like backticks or unescaped quotes) that break the JSON string. Ensure you are using jq to sanitize the input.

2. **Webhook URL is invalid:** Ensure you didn't accidentally copy leading or trailing spaces. The URL must start with `https://`.

3. **Connector Disabled:** Sometimes, IT administrators disable "`Incoming Webhook`" connectors at the tenant level for security reasons. If you cannot find the "Connectors" menu in your channel settings, contact your Microsoft 365 Administrator.

4. **Message size exceeded:** Microsoft Teams has a limit on the size of the webhook payload (`usually around 28KB`). If your stacktrace is massive, you may need to truncate the $INCIDENT_LOGS variable in the bash script before sending it.