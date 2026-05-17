---
icon: material/play-box-multiple
---

# End-to-End Testing Guide

Once you have configured QA Capsule and set up your routing (Slack, Teams, or Jira), it is highly recommended to simulate a pipeline crash to verify the entire telemetry and plugin execution chain.

This guide will walk you through a complete End-to-End (E2E) test.

## Prerequisites

* QA Capsule is running (via `docker-compose up -d`).
* You are logged into the UI as an Administrator.
* You have a terminal open on your machine.

## Phase 1: Provision a Test Endpoint

First, we need to create a project in the system to generate an API key.

* Go to the `CI/CD` Gateways tab in the UI.
* Select GitLab CI.
* Fill in the form:
    * **Project Name** : QA Simulation Project
    * **(Optional) Routing MS Teams** : Add your Teams Webhook.
    * **(Optional) Routing Jira Key** : Add a test Jira key (e.g., TEST).
* Click Provision Project Endpoint.
* Look at the read-only boxes at the bottom. Copy the Generated API Key (it looks like `sre_pk_gitlab_a1b2c3d4`).

## Phase 2: Simulate a CI/CD Crash

Instead of breaking a real pipeline, we will act as the sre-agent.sh and send a manual `HTTP POST` request to the Webhook API using curl.

Open your terminal and paste the following command. Make sure to replace the X-API-Key value with the one you just copied.

```Bash
curl -X POST http://localhost:9000/api/webhooks/gitlab \
     -H "Content-Type: application/json" \
     -H "X-API-Key: sre_pk_gitlab_YOUR_KEY_HERE" \
     -d '{
           "name": "E2E Checkout Workflow",
           "error": "Timeout waiting for element #submit-btn",
           "status": "CRITICAL",
           "console_logs": "[INFO] Starting Cypress...\n[INFO] Navigating to /checkout\n[FATAL] Element #submit-btn not found after 10000ms.\n[TRACE] at Context.eval (checkout.spec.js:42:12)"
         }'
```
If the API key is correct, your terminal will respond with:

```JSON
{"project":"QA Simulation Project","status":"incident_recorded"}
```

## Phase 3: Verify the Dashboard (Real-Time)

1. Switch back to your browser and open the Dashboard tab.
2. Thanks to the real-time polling engine, you should immediately see your Active Incidents KPI increment to 1.
3. The incident will appear in the Incident History Log, styled with a red border because the status is CRITICAL.
4. You will see the exact console_logs formatted perfectly in the code block.

## Phase 4: Verify Plugin Execution

Because the payload contained the word `[FATAL]`, the QA Capsule Plugin Engine will have automatically triggered any active plugins.

1. Go to the Plugin Engine tab.
2. Under the relevant plugin (e.g., Slack or MS Teams), click on the Logs / Console button (or check your Docker terminal logs).
3. You should see the STDOUT confirming the execution:

```Plaintext
    [RUNBOOK] Success (MS Teams Notifier):
    Sending payload to https://company.webhook.office.com/...
    HTTP 200 OK
    [EXIT STATUS] SUCCESS
```
4. Finally, check your actual Slack or Microsoft Teams application. You should have received a beautifully formatted alert card containing the error details!

**Congratulations ! Your SRE Control Plane is fully operational.**