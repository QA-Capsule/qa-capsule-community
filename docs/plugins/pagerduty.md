---
icon: material/bell-ring
---

# PagerDuty

<div align="center" class="integration-hero">
  <img src="../assets/integrations/pagerduty.png" alt="PagerDuty logo">
</div>

Triggers an **Events API v2** event (`event_action: trigger`) to PagerDuty.

| | |
|---|---|
| **Manifest** | `plugins/pagerduty/pagerduty.json` |
| **Type** | `pagerduty` |

---

=== "QA Capsule Side"

    | Variable | Required | Description |
    |----------|-------------|-------------|
    | `PAGERDUTY_ROUTING_KEY` | **Yes** | Event integration / routing key |
    | `PAGERDUTY_API_URL` | No | Default `https://events.pagerduty.com/v2/enqueue` |

    **Gateway**: **PagerDuty Routing Key** field (per-pipeline override).

    **Execute** should return `[PAGERDUTY] Queued (HTTP 200)`.

=== "Provider Side (PagerDuty)"

    ## 1. Service and Events API v2 integration

    1. PagerDuty → **Services** → target on-call service
    2. **Integrations** → add **Events API V2**
    3. Copy the **Integration Key** (= routing key)

    ## 2. Escalation

    Configure escalation policies, schedules, and service filters to reduce noise (QA Capsule sends `severity: critical`).

    ## 3. Test

    ```bash
    curl -X POST https://events.pagerduty.com/v2/enqueue \
      -H "Content-Type: application/json" \
      -d '{"routing_key":"VOTRE_CLE","event_action":"trigger","payload":{"summary":"QA Capsule test","source":"qa-capsule","severity":"critical"}}'
    ```

---

- [Catalog](integrations-catalog.md)
