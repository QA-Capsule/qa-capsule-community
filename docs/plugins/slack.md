---
icon: fontawesome/brands/slack
---

# Slack

<div align="center" class="integration-hero">
  <img src="../assets/integrations/slack.png" alt="Slack logo">
</div>

Sends a formatted alert card to a Slack channel via **Incoming Webhook** (native Go integration, no shell script).

| | |
|---|---|
| **Manifest** | `plugins/slack/slack-notifier.json` |
| **Type** | `slack` |
| **API used** | JSON `POST` to Incoming Webhook URL |

---

=== "QA Capsule Side"

    ## 1. Role prerequisites

    | Action | Minimum role |
    |--------|----------------|
    | Configure secrets / AUTO-RUN | **Manager** or **Platform Admin** |
    | Channel routing per pipeline | **Manager** or **Lead** |
    | Execute (test) | **Lead** |

    ## 2. Variables (Go server)

    | Variable | Required | Where to set | Description |
    |----------|-------------|--------------|-------------|
    | `SLACK_WEBHOOK_URL` | **Yes** | Server env or Plugin **Configure** | URL `https://hooks.slack.com/services/...` |
    | `SLACK_CHANNEL` | No | **CI/CD Gateway** only | Channel per project (e.g. `#alerts-e2e`) |

    ```bash
    export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/T00/B00/XXXX"
    ```

    ## 3. Plugin Engine

    1. Open **Plugin Engine** → **Smart Slack Routing** card
    2. **Configure**: leave `SLACK_WEBHOOK_URL` empty if already in env
    3. **AUTO-RUN**: leave **OFF** until the webhook is tested
    4. **Execute**: should display `[SLACK] Delivered to ... (HTTP 200)`

    Current manifest:

    ```json
    {
      "integration": "slack",
      "name": "Smart Slack Routing",
      "status": "Active",
      "auto_run": true,
      "trigger_on": ["CRITICAL", "Timeout", "ECONNREFUSED", "FLAKY"],
      "config": { "SLACK_WEBHOOK_URL": "" }
    }
    ```

    ## 4. CI/CD Gateway (dynamic routing)

    1. **CI/CD Gateways** → pipeline project
    2. **+ Add configuration** → choose **Smart Slack Routing** (Slack logo)
    3. **Slack Channel** field: `#alerts-frontend` or member ID `U01234567`
    4. Save the gateway

    At runtime, QA Capsule sends `"channel": "<gateway value>"` in the Slack JSON.

    ## 5. Payload sent by QA Capsule

    ```json
    {
      "channel": "#alerts-frontend",
      "attachments": [{
        "color": "#ff4444",
        "title": "SRE Alert: [Playwright] checkout",
        "text": "Error detected by QA Capsule.\n\nTimeout 30000ms",
        "footer": "QA Capsule Remediation Engine"
      }]
    }
    ```

    ## 6. QA Capsule troubleshooting

    | Symptom | Cause | Solution |
    |----------|-------|----------|
    | `[ERROR] SLACK_WEBHOOK_URL not configured` | Missing secret | `export` or Configure |
    | HTTP 404 | Revoked URL | Recreate webhook on Slack side |
    | No message | AUTO-RUN OFF or gateway without Slack | Enable + Add configuration |
    | Wrong channel | Empty `SLACK_CHANNEL` | Fill in gateway |

=== "Provider Side (Slack)"

    ## 1. Create a Slack App

    1. [api.slack.com/apps](https://api.slack.com/apps) → **Create New App** → **From scratch**
    2. Name: `QA Capsule Bot`
    3. Workspace: your organization

    ## 2. Enable Incoming Webhooks

    1. App menu → **Incoming Webhooks** → **On**
    2. **Add New Webhook to Workspace**
    3. Choose a default channel (e.g. `#general`) — QA Capsule can **override** the channel via `SLACK_CHANNEL`
    4. Copy the URL: `https://hooks.slack.com/services/T…/B…/…`

    ## 3. Permissions / best practices

    | Recommendation | Detail |
    |----------------|--------|
    | Dedicated channel | `#sre-qa-alerts` per team or product |
    | No token in Git | URL = secret; rotate if leaked |
    | Invite the app | Target channel must exist; bot invited if private channel |

    ## 4. Slack-side verification

    Manual test:

    ```bash
    curl -X POST -H "Content-Type: application/json" \
      -d '{"text":"QA Capsule test"}' \
      "https://hooks.slack.com/services/VOTRE_URL"
    ```

    Response `ok` → Slack accepts the webhook.

    ## 5. Slack limits

    - Incoming Webhooks: no advanced threads without migrating to `chat.postMessage` API + Bot token (out of community scope)
    - Standard Slack rate limits; large CI bursts → group via QA Capsule correlation

---

## Links

- [Two-sided guide](configuration-guide.md)
- [Integration catalog](integrations-catalog.md)
