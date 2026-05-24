---
icon: material/microsoft-teams
---

# Microsoft Teams

<div align="center" class="integration-hero">
  <img src="../assets/integrations/teams.png" alt="Microsoft Teams logo">
</div>

Posts a **MessageCard** to a Teams Incoming Webhook connector.

| | |
|---|---|
| **Manifest** | `plugins/teams/teams.json` |
| **Type** | `teams` |

---

=== "QA Capsule Side"

    ## Variables

    | Variable | Required | Where |
    |----------|-------------|-----|
    | `TEAMS_WEBHOOK_URL` | **Yes** | Global env **or** gateway field **MS Teams Webhook URL** |

    The engine also accepts the legacy routing alias `TEAMS_WEBHOOK`.

    ## Plugin Engine

    - **Configure**: full connector URL
    - **Execute** → HTTP 200/202 expected
    - **AUTO-RUN**: Manager only

    ## CI/CD Gateway

    **Add configuration** → **Teams** → paste the **channel-specific** connector URL (each team may have its own URL).

    ## Message sent

    Office 365 MessageCard with incident title, status, and error text (markdown).

=== "Provider Side (Microsoft Teams)"

    ## 1. Incoming Webhook connector

    1. Teams → target channel → **⋯** → **Connectors**
    2. Search for **Incoming Webhook** → **Configure**
    3. Name: `QA Capsule Alerts` → create
    4. Copy the URL `https://....webhook.office.com/...`

    ## 2. Best practices

    | Point | Detail |
    |-------|--------|
    | One URL per channel | No dynamic channel routing like Slack — plan one URL per team/project |
    | Security | URL = secret; regenerate if exposed |
    | Enterprise policy | Verify that Incoming Webhooks are allowed by IT |

    ## 3. Test

    ```bash
    curl -H "Content-Type: application/json" -d '{"@type":"MessageCard","@context":"http://schema.org/extensions","summary":"Test","themeColor":"E81123","sections":[{"activityTitle":"QA Capsule test","text":"Hello"}]}' \
      "URL_DU_CONNECTEUR"
    ```

---

- [Configuration Guide](configuration-guide.md)
