---
icon: material/chart-line
---

# Datadog

<div align="center" class="integration-hero">
  <img src="../assets/integrations/datadog.png" alt="Datadog logo">
</div>

Publishes a Datadog **Event** (`alert_type: error`) to the Events API v1.

| | |
|---|---|
| **Manifest** | `plugins/datadog/datadog-event.json` |
| **Type** | `datadog` |

---

=== "QA Capsule Side"

    | Variable | Required | Description |
    |----------|-------------|-------------|
    | `DD_API_KEY` | **Yes** | Datadog API key |
    | `DD_SITE` | No | Default `datadoghq.com` (EU: `datadoghq.eu`) |

    Gateway: optional **Datadog Tags** (e.g. `env:ci,team:checkout`) — future event enrichment.

    URL called: `https://api.{DD_SITE}/api/v1/events` with header `DD-API-KEY`.

=== "Provider Side (Datadog)"

    ## 1. API key

    1. Datadog → **Organization Settings** → **API Keys**
    2. Create a dedicated key `qa-capsule-integration`

    ## 2. Site / region

    | Region | `DD_SITE` |
    |--------|-----------|
    | US | `datadoghq.com` |
    | EU | `datadoghq.eu` |

    ## 3. Dashboards & monitors

    Events appear in **Event Stream**; optional: create a monitor on `source:qa-capsule`.

    ## 4. Test

    ```bash
    curl -X POST "https://api.datadoghq.com/api/v1/events" \
      -H "DD-API-KEY: VOTRE_CLE" -H "Content-Type: application/json" \
      -d '{"title":"QA Capsule test","text":"hello","alert_type":"error"}'
    ```

---

- [Catalog](integrations-catalog.md)
