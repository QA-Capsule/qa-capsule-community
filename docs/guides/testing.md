---
icon: material/play-box-multiple
---

# End-to-End Testing Guide

Validate ingestion, dashboard, integrations, artifacts, flaky detection, and optional CLI/reporter flows.

---

## Prerequisites

* QA Capsule running (`go run ./cmd/qacapsule/main.go` or Docker).
* Logged in as **Platform Admin** (`admin` / `admin` after password change).
* Terminal with `curl` or PowerShell.

---

## Phase 1: Provision a test project

1. **CI/CD Gateways** → create a project (e.g. `QA Simulation Project`).
2. Copy **API Key** and note **Webhook URL** (`http://localhost:9000/api/webhooks/`).

---

## Phase 2: Ingest a failure (enriched JSON)

```bash
curl -X POST http://localhost:9000/api/webhooks/ \
  -H "Content-Type: application/json" \
  -H "X-API-Key: YOUR_KEY" \
  -H "X-Run-Id: e2e-test-1" \
  -d '{
    "name": "E2E Checkout @jira-TEST-1",
    "error": "Timeout waiting for #submit-btn",
    "status": "CRITICAL",
    "browser": "chromium",
    "os": "linux",
    "viewport": "1280x720",
    "execution_time_ms": 4500,
    "console_logs": "[FATAL] Element #submit-btn not found."
  }'
```

Expected `202` body:

```json
{
  "status": "success",
  "failures_processed": 1,
  "last_incident_id": 1,
  "incident_ids": [1]
}
```

---

## Phase 3: Dashboard

1. Open **Dashboard** → incident appears (red / CRITICAL).
2. Confirm name, logs, and project filter.

---

## Phase 4: Plugin / integration execution

1. **Plugin Engine** → ensure Slack/Jira/Teams integration is **Active**.
2. Set secrets on the server (`SLACK_WEBHOOK_URL`, etc.) — see [Plugin overview](../plugins/overview.md).
3. After ingestion, check server logs for `remediation auto-trigger` or run **Execute** manually on a card.

---

## Phase 5: Artifact upload

```bash
# Create a small zip for testing
zip trace.zip README.md   # Linux/macOS

curl -X POST "http://localhost:9000/api/incidents/1/artifacts" \
  -H "X-API-Key: YOUR_KEY" \
  -F "file=@trace.zip"
```

Expect `202` + file under `data/artifacts/incident_1/`.

---

## Phase 6: Flaky detection

1. **Resolve** the incident in the UI.
2. Re-send the **same** JSON (same `name` + `error`, new `X-Run-Id`).
3. New incident name should start with `[FLAKY]`.

Check API:

```bash
# Fingerprint = SHA256("E2E Checkout @jira-TEST-1|Timeout waiting for #submit-btn")
curl "http://localhost:9000/api/incidents/check-flaky/YOUR_64_CHAR_HASH"
```

---

## Phase 7: Performance regression (optional)

Send three fast passes, then one slow pass (same test name):

```bash
for i in 1 2 3; do
  curl -s -X POST http://localhost:9000/api/webhooks/ \
    -H "X-API-Key: YOUR_KEY" -H "X-Run-Id: perf-$i" \
    -H "Content-Type: application/json" \
    -d '{"name":"Perf login","status":"PASSED","error":"","execution_time_ms":1000}'
done

curl -X POST http://localhost:9000/api/webhooks/ \
  -H "X-API-Key: YOUR_KEY" -H "X-Run-Id: perf-slow" \
  -H "Content-Type: application/json" \
  -d '{"name":"Perf login","status":"PASSED","error":"","execution_time_ms":3500}'
```

Expect `[PERF] Perf login` incident.

---

## Phase 8: Developer CLI (optional)

```bash
go build -o bin/qacapsule-cli ./cmd/cli
export QACAPSULE_API_URL=http://localhost:9000
export QACAPSULE_API_KEY=YOUR_KEY

bin/qacapsule-cli run --test-name "E2E Checkout @jira-TEST-1" \
  --test-error "Timeout waiting for #submit-btn" \
  cmd /c "exit 1"
```

Yellow flaky warning appears if Phase 6 succeeded.

---

## Phase 9: Playwright reporter (optional)

See [Playwright Reporter](../integration/playwright-reporter.md).

---

## Success criteria

| Check | Pass |
|---|---|
| Webhook returns `202` + `last_incident_id` | ☐ |
| Dashboard shows incident | ☐ |
| Integration logs / notification | ☐ |
| Artifact on disk | ☐ |
| `[FLAKY]` after re-fail | ☐ |
| `[PERF]` after slow pass (optional) | ☐ |

**Your SRE control plane is operational.**
