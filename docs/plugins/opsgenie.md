---
icon: material/bell-alert
---

# Opsgenie (Atlassian)

<div align="center" class="integration-hero">
  <img src="../assets/integrations/opsgenie.png" alt="Opsgenie logo">
</div>

Creates an alert via `POST /v2/alerts` (header `Authorization: GenieKey …`).

| | |
|---|---|
| **Manifest** | `plugins/opsgenie/opsgenie-alert.json` |
| **Type** | `opsgenie` |

---

=== "QA Capsule Side"

    | Variable | Required | Default |
    |----------|-------------|--------|
    | `OPSGENIE_API_KEY` | **Yes** | — |
    | `OPSGENIE_API_URL` | No | `https://api.opsgenie.com` |

    Gateway: **Opsgenie Team** (optional, future metadata).

    Priority sent: `P1`. Message = incident summary + description = test error.

=== "Provider Side (Opsgenie)"

    ## 1. API key

    1. Opsgenie → **Settings** → **Integrations** → **API**
    2. Create a key with **Create and Update Alerts** permission
    3. Copy the `GenieKey …` key

    ## 2. Teams and routing

    Associate the key with the on-call team; define escalation policies in Opsgenie.

    ## 3. Test

    ```bash
    curl -X POST https://api.opsgenie.com/v2/alerts \
      -H "Authorization: GenieKey VOTRE_CLE" \
      -H "Content-Type: application/json" \
      -d '{"message":"QA Capsule test","description":"test","priority":"P3"}'
    ```

---

- [Catalog](integrations-catalog.md)
