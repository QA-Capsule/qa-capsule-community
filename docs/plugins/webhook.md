---
icon: material/webhook
---

# Custom HTTP Webhook

<div align="center" class="integration-hero">
  <img src="../assets/integrations/webhook.png" alt="Webhook">
</div>

Generic JSON `POST` to your internal API (ServiceNow, runbook, custom orchestrator).

| | |
|---|---|
| **Manifest** | `plugins/webhook/custom-webhook.json` |
| **Type** | `webhook` |

Also used for **QA flaky report**, **TestRail**, **Zephyr**, **Xray** (same HTTP runner).

---

=== "QA Capsule Side"

    | Variable | Required | Description |
    |----------|-------------|-------------|
    | `WEBHOOK_URL` | **Yes** | Target HTTPS URL |
    | `WEBHOOK_AUTH_HEADER` | No | E.g. `Bearer xxx` (Authorization header) |

    **Gateway**: **Custom Webhook URL** per project.

    ## Default payload

    ```json
    {
      "source": "qa-capsule",
      "event": "incident.detected",
      "incident": "test name",
      "error": "message",
      "status": "CRITICAL",
      "action": "AUTO_EVENT:..."
    }
    ```

    Your API must respond **2xx** for success on the Plugin Engine side.

=== "Provider Side (your API)"

    ## 1. Receiver endpoint

    - Method **POST**, `Content-Type: application/json`
    - TLS recommended (HTTPS)
    - Auth: API key, mTLS, or IP allowlist for the QA Capsule server

    ## 2. Idempotence

    Plan for `fingerprint` or `incident_id` in a custom extension to avoid duplicates.

    ## 3. Test

    ```bash
    curl -X POST https://votre-api.internal/hooks/qa-capsule \
      -H "Content-Type: application/json" \
      -d '{"source":"qa-capsule","event":"test"}'
    ```

---

- [Catalog](integrations-catalog.md)
