---
icon: material/puzzle-outline
---

# How Plugins Work (The Plugin Engine)

The true power of QA Capsule lies in its automation capabilities. Instead of hardcoding integrations (like Jira, Slack, Teams, or PagerDuty) directly into the Go backend, QA Capsule uses a modular, asynchronous **Plugin Engine**.

This engine allows Site Reliability Engineers (SREs) and DevOps teams to write custom remediation scripts in Bash or Python. QA Capsule evaluates incoming incidents and automatically triggers these scripts, injecting dynamic context along the way.

---

## 1. The Plugin Architecture

Every plugin in QA Capsule is an independent folder residing inside the `plugins/` directory of the host server. 

A standard plugin requires exactly two files to function:

1. **The Manifest (`manifest.json`):** A configuration file that tells QA Capsule what the plugin is, what global variables it needs, and when to trigger it.
2. **The Executable (`exec.sh` or `exec.py`):** The actual script that performs the external action (e.g., making an API call to Slack or Jira).

### Directory Structure Example
```text
qa-capsule/
├── internal/
├── data/
└── plugins/
    ├── slack-notifier/
    │   ├── manifest.json
    │   └── slack.sh
    ├── jira-creator/
    │   ├── manifest.json
    │   └── jira.sh
    └── custom-reboot-script/
        ├── manifest.json
        └── reboot.py
```

## 2. The manifest.json File

The manifest is the bridge between the QA Capsule UI and your raw script. It dictates how the plugin appears in the Dashboard and how the Go backend should treat it.
Example Manifest

```JSON
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

Manifest Fields Explained:

* `name & description:` Displayed on the Plugin Card in the UI.
* `entrypoint:` The file QA Capsule should execute (must have executable permissions: chmod +x).
* `global_env:` An array of strings. These are variables that apply to the entire system (like an authentication token). When you define these here, a "Configure" button appears in the UI, allowing Administrators to fill in these values securely.
* `trigger_on:` An array of trigger keywords. If an incoming incident's console_logs or error field contains ANY of these exact strings, the plugin will execute automatically.

## 3. Dynamic Context Injection (The Magic)

You might wonder: "If I only have one Slack plugin, how does it know to send Frontend errors to the #frontend-team channel, and Backend errors to the #backend-team channel?"

This is solved by Dynamic Context Injection.

When a webhook receives a payload, it identifies the project using the X-API-Key. Before executing your plugin script, the Go backend pulls the routing data for that specific project from the SQLite database and injects it into the script as Environment Variables.
Available Environment Variables inside your script:

### A. Incident Data (The Payload)

* `INCIDENT_NAME`: The title of the failed test.

* `INCIDENT_ERROR`: The short error summary.

* `INCIDENT_STATUS`: Usually CRITICAL or WARNING.

* `INCIDENT_LOGS`: The raw, multi-line stacktrace.

* `INCIDENT_BROWSER`: The environment context.

### B. Project Routing Data (From SQLite)

* `SLACK_CHANNEL`: The specific Slack channel defined in the CI/CD Gateway UI for this project.

* `JIRA_PROJECT_KEY`: The specific Jira board prefix (e.g., PAY).

* `TEAMS_WEBHOOK`: The specific MS Teams webhook URL for the project's channel.

### C. Global Plugin Data

* Any variables defined in your global_env array inside manifest.json (e.g., `SLACK_WEBHOOK_URL`).

## 4. Writing the Executable Script

Because all data is passed as environment variables, writing the plugin script is incredibly straightforward. You do not need to parse JSON or query a database.
Example: slack.sh

```bash
#!/bin/bash
# ==============================================================================
# QA Capsule Plugin: Slack Notifier
# ==============================================================================

# 1. Check if the required global token is configured in the UI
if [ -z "$SLACK_WEBHOOK_URL" ]; then
  echo "[ERROR] SLACK_WEBHOOK_URL is not configured in the Plugin UI."
  exit 1
fi

# 2. Check if the project actually has a Slack channel assigned
if [ -z "$SLACK_CHANNEL" ]; then
  echo "[SKIP] No SLACK_CHANNEL defined for this project. Aborting."
  exit 0
fi

# 3. Format the Slack Payload
# We use jq to safely escape the multi-line stacktrace for JSON
SAFE_LOGS=$(echo "$INCIDENT_LOGS" | jq -R -s '.')

JSON_PAYLOAD=$(cat <<EOF
{
  "channel": "$SLACK_CHANNEL",
  "attachments": [
    {
      "color": "#FF0000",
      "title": "QA Capsule Alert: $INCIDENT_NAME",
      "text": "*Error:* $INCIDENT_ERROR\n*Environment:* $INCIDENT_BROWSER\n\n*Stacktrace:*\n\`\`\`\n$INCIDENT_LOGS\n\`\`\`"
    }
  ]
}
EOF
)

# 4. Execute the API Call
echo "[INFO] Pushing alert to Slack channel: $SLACK_CHANNEL..."

HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST -H 'Content-type: application/json' --data "$JSON_PAYLOAD" "$SLACK_WEBHOOK_URL")

if [ "$HTTP_STATUS" -eq 200 ]; then
  echo "[SUCCESS] Slack message delivered."
  exit 0
else
  echo "[ERROR] Failed to send Slack message. HTTP $HTTP_STATUS"
  exit 1
fi
```

## 5. Execution and Debugging
### Asynchronous Execution

Plugins execute asynchronously in the background. This guarantees that even if Jira's API is down and the script hangs for 30 seconds, your CI/CD pipeline is not delayed waiting for the QA Capsule webhook to respond.

### Capturing StdOut / StdErr

The Go backend captures everything your script prints to `stdout` (`echo`, `print()`) and `stderr`.

If your script fails or behaves unexpectedly, you can view the live execution logs directly in the QA Capsule UI:

* Navigate to the Plugin Engine tab.
* Click on the Terminal Icon (Stdout Logs) on your specific plugin card.
* You will see exactly what your Bash or Python script printed during its last run, including curl HTTP codes or syntax errors.

### Manual Execution

While plugins auto-run based on the `trigger_on` array, Administrators can manually test them at any time by clicking the Execute button on the plugin card. Note that manual execution will run the script without project-specific context variables (like `SLACK_CHANNEL`), so ensure your script handles empty variables gracefully.