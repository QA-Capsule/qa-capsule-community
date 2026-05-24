---
icon: fontawesome/brands/jira
---

# Jira Software

<div align="center" class="integration-hero">
  <img src="../assets/integrations/jira.png" alt="Jira logo">
</div>

Automatically creates a Jira ticket (Bug / Task) via the REST API `POST /rest/api/2/issue`.

| | |
|---|---|
| **Manifest** | `plugins/jira/jira-ticket.json` |
| **Type** | `jira` |
| **Auth** | Basic (email + Atlassian API token) |

---

=== "QA Capsule Side"

    ## Server variables

    | Variable | Required | QA Capsule source |
    |----------|-------------|-------------------|
    | `JIRA_URL` | **Yes** | Env / Configure (e.g. `https://company.atlassian.net`) |
    | `JIRA_EMAIL` | **Yes** | Atlassian service account |
    | `JIRA_API_TOKEN` | **Yes** | Atlassian token (never in plain text in Git) |
    | `JIRA_ISSUE_TYPE` | No | Default `Bug` |
    | `JIRA_PROJECT_KEY` | **Yes** for auto | **CI/CD Gateway** or `@SCRUM-42` tag in test name |

    ```bash
    export JIRA_URL="https://votre-domaine.atlassian.net"
    export JIRA_EMAIL="sre-bot@company.com"
    export JIRA_API_TOKEN="xxxxxxxx"
    ```

    ## Plugin Engine

    1. **Configure**: optional if everything is in env
    2. **Execute** without `JIRA_PROJECT_KEY` → explicit error (intended behavior)
    3. **AUTO-RUN ON** only after a successful Execute

    ## CI/CD Gateway

    - **Add configuration** → **Jira Auto-Ticketing**
    - **Jira Project Key** field: `SCRUM`, `PAY`, etc.

    ## Automatic extraction from test

    Test name containing `@jira-SCRUM-99` or `@SCRUM-99` → issue / project key injected at ingestion.

    ## Created payload

    ```json
    {
      "fields": {
        "project": { "key": "SCRUM" },
        "summary": "[QA Capsule] checkout payment",
        "description": "Incident from QA Capsule.\n\nTimeout...",
        "issuetype": { "name": "Bug" }
      }
    }
    ```

    ## Troubleshooting

    | Error | Solution |
    |--------|----------|
    | HTTP 401 | Incorrect token or email |
    | HTTP 400 project | Invalid `JIRA_PROJECT_KEY` |
    | No ticket | AUTO-RUN off or Jira missing from gateway |

=== "Provider Side (Atlassian Jira)"

    ## 1. Service account

    1. Create a dedicated user (e.g. `sre-bot@company.com`) or use a bot account approved by the Jira admin
    2. Invite them to monitored projects with **Create issues** permission

    ## 2. API Token

    1. [id.atlassian.com](https://id.atlassian.com) → **Security** → **API tokens**
    2. **Create API token** → copy once
    3. Store in secret manager / `export JIRA_API_TOKEN` on the QA Capsule server

    ## 3. Jira project

    | Element | Where to find it |
    |---------|----------------|
    | **Project Key** | Project settings → e.g. `SCRUM` |
    | **Issue type** | Project scheme (Bug, Task, Story) → align `JIRA_ISSUE_TYPE` |

    ## 4. Atlassian API test

    ```bash
    curl -u "email:API_TOKEN" -H "Content-Type: application/json" \
      -d '{"fields":{"project":{"key":"SCRUM"},"summary":"QA Capsule test","issuetype":{"name":"Task"}}}' \
      "https://VOTRE-DOMAINE.atlassian.net/rest/api/2/issue"
    ```

    HTTP **201** + `id` → provider configuration OK.

---

- [Configuration Guide](configuration-guide.md) · [Catalog](integrations-catalog.md)
