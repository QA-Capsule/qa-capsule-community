---
icon: material/phone-alert
---

# VictorOps / Splunk On-Call

<div align="center" class="integration-hero">
  <img src="../assets/integrations/victorops.png" alt="Splunk On-Call logo">
</div>

Sends a **CRITICAL** incident to the Splunk On-Call (formerly VictorOps) REST routing URL.

| | |
|---|---|
| **Manifest** | `plugins/victorops/victorops-alert.json` |
| **Type** | `victorops` |

---

=== "QA Capsule Side"

    | Variable | Required |
    |----------|-------------|
    | `VICTOROPS_ROUTING_URL` | **Yes** (full REST URL provided by Splunk) |

    Configurable per pipeline via gateway **VictorOps Routing URL**.

    JSON body: `message_type: CRITICAL`, `entity_display_name: QA Capsule`, `state_message` with incident summary.

=== "Provider Side (Splunk On-Call)"

    ## 1. Obtain the routing URL

    1. Splunk On-Call → **Settings** → integration / REST endpoint
    2. Copy the unique POST URL (often per team)

    ## 2. Alert policy

    Associate the endpoint with on-call rotation; adjust thresholds to avoid CI flood.

    ## 3. Test

    Post minimal JSON with `message_type` + `state_message` to the provided URL (see Splunk On-Call documentation for the exact schema for your tenant).

---

- [Catalog](integrations-catalog.md)
