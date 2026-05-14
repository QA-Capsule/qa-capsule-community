---
icon: fontawesome/brands/slack
---

# Slack Integration

The Slack plugin ensures that your engineering teams are instantly notified when a pipeline crashes. Instead of relying on generic CI/CD bot messages that cause "alert fatigue," QA Capsule formats the alert with contextual data, color-coded urgency, and the exact StackTrace.

More importantly, thanks to **Dynamic Routing**, a single Slack integration can route Frontend errors to `#alerts-frontend` and Backend errors to `#alerts-backend` automatically.

---

## Part 1: Slack Workspace Configuration

To send messages to Slack, QA Capsule uses **Incoming Webhooks**. You need to create a lightweight Slack App in your workspace to generate this webhook URL.

### Step 1: Create the Slack App
1. Navigate to the [Slack API Apps Portal](https://api.slack.com/apps) in your web browser.
2. Click the **Create New App** button.
3. Select **From scratch**.
4. **App Name:** `QA Capsule Bot` (or similar).
5. **Workspace:** Select your company's Slack workspace.
6. Click **Create App**.

### Step 2: Enable Incoming Webhooks
1. In your new App's settings menu (left sidebar), click on **Incoming Webhooks**.
2. Toggle the switch to **Activate Incoming Webhooks** (turn it On).
   *(Placeholder: `![Enable Slack Webhooks](../assets/slack-enable-webhook.png)`)*
3. Scroll down and click **Add New Webhook to Workspace**.
4. Slack will ask you to pick a channel. Pick any default channel (e.g., `#general` or `#test`). **Do not worry about this choice**—QA Capsule will dynamically override this channel later based on the project routing rules.
5. Click **Allow**.
6. **CRITICAL:** Copy the generated **Webhook URL** (it will look like `https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX`).

---

## Part 2: QA Capsule Global Configuration

Now we must provide the global Slack Webhook URL to the QA Capsule Plugin Engine.

1. Open your **QA Capsule Dashboard**.
2. Navigate to the **Plugin Engine** tab.
3. Locate the **Slack Alert Notifier** plugin card.
4. Click the **Configure** button.
5. Paste the URL you copied in Step 1 into the `SLACK_WEBHOOK_URL` field.
6. Click **Save Configuration**.
7. Ensure the plugin's status switch is toggled to **AUTO-RUN ON**.

---

## Part 3: Project-Level Dynamic Routing

Because QA Capsule is multi-tenant, you must define which Slack channel receives alerts for which project.

1. Navigate to the **CI/CD Gateways** module.
2. When provisioning a new endpoint (or editing an existing one), locate the **Routing: Slack Channel** field.

3. Enter the exact name of your channel, including the hash (e.g., `#alerts-frontend`).
   * *Note: If routing to a specific user via Direct Message, use their Member ID (e.g., `@U12345678`).*
   
4. Click **Provision Project Endpoint**.

When this specific project fails, QA Capsule will inject `SLACK_CHANNEL="#alerts-frontend"` into the plugin script environment.

---

## Part 4: The Plugin Script Implementation (`slack.sh`)

For SREs who want to customize the look and feel of the Slack message, here is the underlying Bash script used by the plugin. 

It uses Slack's `attachments` array to create a visually distinct, red-bordered card for critical incidents.

### The `manifest.json`
```json
{
  "name": "Slack Alert Notifier",
  "description": "Pushes formatted incident alerts to a specific Slack channel.",
  "version": "1.0.0",
  "author": "SRE Team",
  "entrypoint": "slack.sh",
  "global_env": [
    "SLACK_WEBHOOK_URL"
  ],
  "trigger_on": [
    "[FATAL]",
    "CRITICAL"
  ]
}
```

### The `slack.sh` Executable

``` Bash
#!/bin/bash
# ==============================================================================
# QA Capsule Plugin: Slack Notifier
# ==============================================================================

# 1. Validate Global Authentication
if [ -z "$SLACK_WEBHOOK_URL" ]; then
  echo "[ERROR] SLACK_WEBHOOK_URL is not configured globally. Aborting."
  exit 1
fi

# 2. Validate Dynamic Routing Context
if [ -z "$SLACK_CHANNEL" ]; then
  echo "[SKIP] No SLACK_CHANNEL defined for this project. Skipping Slack notification."
  exit 0
fi

echo "[INFO] Formatting Slack payload for channel: $SLACK_CHANNEL..."

# 3. Safely escape multi-line stacktraces and errors using jq
# This prevents bash injection and malformed JSON errors
SAFE_ERROR=$(echo "$INCIDENT_ERROR" | jq -R -s '.')
SAFE_LOGS=$(echo "$INCIDENT_LOGS" | jq -R -s '.')
SAFE_BROWSER=$(echo "$INCIDENT_BROWSER" | jq -R -s '.')

# Determine color based on Incident Status
if [ "$INCIDENT_STATUS" == "CRITICAL" ]; then
  COLOR="#FF0000" # Red
else
  COLOR="#FFCC00" # Yellow for warnings
fi

# 4. Construct the Slack JSON Payload
# Note: We use the "channel" parameter to dynamically override the webhook's default channel
JSON_PAYLOAD=$(cat <<EOF
{
  "channel": "$SLACK_CHANNEL",
  "username": "QA Capsule Bot",
  "icon_emoji": ":rotating_light:",
  "attachments": [
    {
      "color": "$COLOR",
      "pretext": "*New CI/CD Failure Detected*",
      "title": "$INCIDENT_NAME",
      "fields": [
        {
          "title": "Error Summary",
          "value": ${SAFE_ERROR},
          "short": false
        },
        {
          "title": "Environment / Context",
          "value": ${SAFE_BROWSER},
          "short": true
        },
        {
          "title": "Status",
          "value": "$INCIDENT_STATUS",
          "short": true
        }
      ],
      "text": "*Console Logs & StackTrace:*\n\`\`\`\n${INCIDENT_LOGS:1:-1}\n\`\`\`",
      "footer": "QA Flight Recorder",
      "ts": $(date +%s)
    }
  ]
}
EOF
)

# 5. Execute the HTTP Request via cURL
echo "[INFO] Dispatching to Slack API..."

HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST -H 'Content-type: application/json' --data "$JSON_PAYLOAD" "$SLACK_WEBHOOK_URL")

if [ "$HTTP_STATUS" -eq 200 ]; then
  echo "[SUCCESS] Alert successfully delivered to $SLACK_CHANNEL."
  exit 0
else
  echo "[ERROR] Failed to send Slack message. HTTP Status: $HTTP_STATUS"
  exit 1
fi
```
## Part 5: Troubleshooting

If you are not receiving Slack messages, check the Stdout Logs terminal in the QA Capsule Plugin Engine UI.

1. **HTTP 404 Not Found (channel_not_found):** The `SLACK_CHANNEL` specified in your CI/CD Gateways routing does not exist, or it is misspelled.

2. **HTTP 403 Forbidden (action_prohibited):** If you are trying to send a message to a Private Channel, the Slack App will fail unless you manually invite the App to that channel first. Go to the private channel in Slack, type /invite `@QA Capsule Bot`, and hit enter.

3. **HTTP 400 Bad Request:** The payload is malformed. This usually happens if a stacktrace contains massive, unescaped unicode characters that jq struggles with.