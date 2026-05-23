---
icon: material/webhook
---

# Webhooks API Reference

Primary ingestion gateway for CI/CD telemetry: **JSON** (single or batch) and **JUnit XML** upload.

---

## Asynchronous processing

Valid requests are persisted and return **`202 Accepted`** immediately. Correlation, perf analysis, and plugin execution run in background goroutines.

| Environment | Base URL |
|---|---|
| Local / Docker | `http://localhost:9000` |
| Production | `https://sre.yourcompany.com` |

---

## Authentication

```http
X-API-Key: <project_api_key>
```

Optional headers:

| Header | Description |
|---|---|
| `X-Run-Id` or `X-Pipeline-Run-Id` | Groups deduplication to one pipeline execution (recommended) |

---

## JSON endpoint

```
POST /api/webhooks/
POST /api/webhooks/{provider}
```

`{provider}` is a logical label (`github`, `gitlab`, `custom`, …) for your logs only.

---

## Unified alert schema

### Single event

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes* | Test or job title (*or use batch `tests`) |
| `status` | string | Yes | `FAILED`, `CRITICAL`, `PASSED`, `WARNING`, … |
| `error` | string | Yes** | Error summary (**empty allowed for `PASSED`) |
| `console_logs` | string or array | No | STDOUT / context |
| `browser` | string | No | Browser or runner label |
| `os` | string | No | OS dimension (`linux`, `windows`, …) |
| `viewport` | string | No | Viewport (`1280x720`, …) |
| `execution_time_ms` | number | No | Test duration in milliseconds |
| `jira_issue_key` | string | No | Explicit Jira key; else parsed from `name` |
| `framework` | string | No | `playwright`, `robotframework`, or default |

### Batch mode

```json
{
  "tests": [
    {
      "name": "Checkout @jira-SCRUM-42",
      "status": "FAILED",
      "error": "Timeout 30000ms",
      "browser": "chromium",
      "os": "linux",
      "viewport": "1280x720",
      "execution_time_ms": 5200
    }
  ]
}
```

### Playwright-shaped payload

When `framework` is `playwright`:

| Field | Maps to |
|---|---|
| `title` | Incident name |
| `failure_reason` | Error |
| `project` | Browser context (unless `browser` set) |
| `execution_time_ms` | Duration |

---

## Jira tags in test names

The ingestion layer extracts issue keys from test titles:

| Pattern in `name` | Extracted key |
|---|---|
| `@jira-SCRUM-42` | `SCRUM-42` |
| `@SCRUM-42` | `SCRUM-42` |

Extracted keys are stored on the incident and passed to the Jira integration (`JIRA_ISSUE_KEY`, `JIRA_PROJECT_KEY`).

---

## Performance regression (`PASSED`)

For `status: PASSED` with `execution_time_ms`:

1. Samples are stored in `test_execution_metrics`.
2. If current duration **> 150%** of the 30-day average for the same fingerprint, an incident is created:
   - Name: `[PERF] <original name>`
   - Status: `PERF_DEGRADATION`

Failed tests always create incidents (subject to dedup rules).

---

## Example — cURL

```bash
curl -f -X POST "http://localhost:9000/api/webhooks/" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: YOUR_PROJECT_KEY" \
  -H "X-Run-Id: build-1234" \
  -d '{
    "name": "Login flow @jira-SCRUM-99",
    "status": "FAILED",
    "error": "Timeout 30000ms exceeded",
    "browser": "chromium",
    "os": "windows",
    "viewport": "1280x720",
    "execution_time_ms": 5000,
    "console_logs": "Waiting for selector..."
  }'
```

---

## Response `202 Accepted`

```json
{
  "status": "success",
  "failures_processed": 1,
  "incident_ids": [42],
  "last_incident_id": 42,
  "project": "Frontend E2E",
  "pipeline_run_id": "build-1234"
}
```

Use `last_incident_id` to upload artifacts (traces, screenshots).

---

## JUnit XML upload

```
POST /api/webhooks/upload?framework=Playwright
Content-Type: multipart/form-data
```

Form field: `file` (XML report). See [JUnit XML Upload](../integration/junit-xml-upload.md).

---

## HTTP errors

| Code | Meaning |
|---|---|
| `400` | Invalid JSON or multipart |
| `401` | Missing `X-API-Key` |
| `403` | Unknown API key |

---

## Related

- [Incidents API](incidents-api.md) — artifacts, flaky check
- [Playwright Reporter](../integration/playwright-reporter.md)
- [Incident Lifecycle](../guides/incident-lifecycle.md)
