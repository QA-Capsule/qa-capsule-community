---
icon: material/clipboard-check
---

# Test management (TestRail, Zephyr, Xray)

<div align="center" class="integration-hero">
  <img src="../assets/integrations/testrail.png" alt="TestRail" style="margin-right:16px">
  <img src="../assets/integrations/zephyr.png" alt="Zephyr" style="margin-right:16px">
  <img src="../assets/integrations/xray.png" alt="Xray">
</div>

| Tool | Manifest | Type |
|-------|----------|------|
| TestRail | `testrail/testrail-result.json` | `testrail` |
| Zephyr Scale | `zephyr/zephyr-execution.json` | `zephyr` |
| Xray Cloud | `xray/xray-result.json` | `xray` |

The community engine uses the **webhook runner**: configure a receiver URL per tool.

---

=== "TestRail"

    === "QA Capsule Side"

        | Variable | Usage |
        |----------|--------|
        | `WEBHOOK_URL` | URL of your bridge or middleware |
        | Gateway | **TestRail Webhook URL** |

        Variables documented for a future native API:

        | Variable | Description |
        |----------|-------------|
        | `TESTRAIL_URL` | Instance |
        | `TESTRAIL_USER` | API user |
        | `TESTRAIL_API_KEY` | API key |
        | `TESTRAIL_RUN_ID` | Active run |
        | `TESTRAIL_CASE_ID` | Failing case |

    === "Provider Side (TestRail)"

        1. TestRail → user profile → **API Key**
        2. Create a test run; note **Run ID** and **Case ID**
        3. Option: middleware (Azure Function, small service) that receives the QA Capsule JSON and calls the TestRail `add_result_for_case` API

=== "Zephyr Scale"

    === "QA Capsule Side"

        Gateway: **Zephyr Webhook URL** → effective `WEBHOOK_URL`.

        Future variables: `ZEPHYR_API_TOKEN`, `ZEPHYR_PROJECT_KEY`, `ZEPHYR_TEST_CASE_KEY`.

    === "Provider Side (Zephyr)"

        1. Jira → Zephyr Scale → **API Keys**
        2. Document test case keys (`PROJ-T42`)
        3. Webhook or Jira automation to mark execution as failed

=== "Xray Cloud"

    === "QA Capsule Side"

        Gateway: **Xray Webhook URL**.

        Future variables: `XRAY_CLIENT_ID`, `XRAY_CLIENT_SECRET`, `XRAY_TEST_KEY`.

    === "Provider Side (Xray)"

        1. Xray Cloud → **API Keys** (OAuth client)
        2. Associate Jira tests; use execution keys
        3. Middleware recommended until native Xray runner

---

## QA Capsule payload received by your webhook

```json
{
  "source": "qa-capsule",
  "event": "incident.detected",
  "incident": "[Playwright] login",
  "error": "assertion failed",
  "status": "CRITICAL"
}
```

---

- [Webhook](webhook.md) · [Catalog](integrations-catalog.md)
