# Webhooks API Reference

The Webhooks API is the primary ingestion gateway for CI/CD telemetry. It accepts both **JSON payloads** (pipeline-level alerts) and is complemented by the [JUnit XML Upload](../integration/junit-xml-upload.md) endpoint for structured test reports.

---

## Core Concepts

### Asynchronous Processing

The API is **asynchronous**. Valid requests are persisted to SQLite and return `202 Accepted` immediately. Parsing, routing, and plugin execution happen in background goroutines — your CI pipeline is never blocked.

### Base URL

| Environment | URL |
|---|---|
| Local / Docker | `http://localhost:9000` |
| Production | `https://sre.yourcompany.com` |

---

## Authentication

Every webhook request requires a **Project API Key** in the `X-API-Key` header:

```http
POST /api/webhooks/custom HTTP/1.1
Host: sre.yourcompany.com
Content-Type: application/json
X-API-Key: sre_pk_custom_9f8a7b6c5d4e3f2a1
```

The key identifies the project and loads routing variables (Slack channel, Jira key, Teams webhook).

---

## JSON Webhook Endpoint

```
POST /api/webhooks/{provider}
```

| Path param | Values | Description |
|---|---|---|
| `provider` | `github`, `gitlab`, `jenkins`, `custom` | Logical source label for logging |

---

## Unified Alert Schema

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Incident title — test name or job name (max 255 chars) |
| `status` | string | Yes | `CRITICAL`, `WARNING`, or `INFO` |
| `error` | string | Yes | One-line error summary (max 500 chars) |
| `console_logs` | string | Yes | Detailed logs, stack trace, stdout (use `\n` for newlines) |
| `browser` | string | No | Context metadata — OS, browser, runner image |

### Status definitions

| Status | Behavior |
|---|---|
| `CRITICAL` | Triggers plugin engine (Slack, Teams, Jira) |
| `WARNING` | Logged in dashboard only — no notifications |
| `INFO` | Recorded unless plugins explicitly listen for it |

---

## Example — cURL

```bash
curl -f -X POST "https://sre.yourcompany.com/api/webhooks/custom" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: sre_pk_custom_a1b2c3d4" \
  -d '{
    "name": "Nightly Security Scan",
    "status": "CRITICAL",
    "error": "Vulnerability threshold exceeded: 3 High severity issues.",
    "browser": "Docker/Trivy",
    "console_logs": "[FATAL] Scan failed.\n\nCVE-2023-1234 found in package log4j."
  }'
```

---

## Example — Node.js

```javascript
async function sendToQACapsule(testResult) {
  const response = await fetch(`${process.env.QA_CAPSULE_URL}/api/webhooks/custom`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': process.env.QA_CAPSULE_API_KEY,
    },
    body: JSON.stringify({
      name: testResult.testName,
      status: testResult.isFlaky ? 'WARNING' : 'CRITICAL',
      error: testResult.failureMessage.substring(0, 200),
      browser: process.env.CI_ENVIRONMENT || 'local',
      console_logs: `[FATAL] Test failed.\n\n--- STACKTRACE ---\n${testResult.stackTrace}`,
    }),
  });

  if (response.status === 202) {
    console.log('Telemetry accepted by QA Capsule.');
  } else {
    console.error(`QA Capsule error: HTTP ${response.status}`);
  }
}
```

---

## HTTP Responses

### `202 Accepted`

```json
{
  "project": "Security Audits",
  "status": "incident_recorded",
  "timestamp": "2026-05-17T14:32:01Z"
}
```

### `400 Bad Request`

Malformed JSON or missing required fields.

### `401 Unauthorized`

Missing `X-API-Key` header.

### `403 Forbidden`

API key not found in database (revoked or never provisioned).

---

## JUnit XML Upload (Recommended)

For structured multi-test reports, use the dedicated upload endpoint instead:

```
POST /api/webhooks/upload?framework=Playwright
```

See [JUnit XML Upload](../integration/junit-xml-upload.md) for full documentation.

---

## Related

- [CI/CD Integration Overview](../integration/cicd-overview.md)
- [Incident Lifecycle](../guides/incident-lifecycle.md)
- [Incidents REST API](incidents-api.md)
