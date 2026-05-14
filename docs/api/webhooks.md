---
icon: material/api
---

# Webhooks API Reference

The QA Capsule Webhooks API is the primary ingestion gateway for all telemetry, incident reports, and test failures. It is designed to be highly available, asynchronous, and agnostic to the testing framework you are using.

If you are using a CI/CD provider that does not have a native QA Capsule integration guide, or if you want to push custom metrics from your internal tools, you can interact directly with this API.

---

## 1. Core Concepts

### Asynchronous Processing
The Webhooks API is **asynchronous**. When you send a payload to the API, QA Capsule immediately validates the JSON schema and the API Key. If valid, it saves the raw data to the SQLite database and returns a `202 Accepted` response. 

The heavy lifting—parsing the StackTrace, evaluating routing rules, and triggering bash plugins (like Slack or Jira)—happens in a background worker thread. This ensures that the QA Capsule API never blocks or slows down your CI/CD pipelines.

### Base URL
Depending on your deployment, your base URL will look like this:
* **Local/Docker:** `http://localhost:9000`
* **Production:** `https://sre.yourcompany.com`

---

## 2. Authentication

Every request to the Webhooks API **must** be authenticated using a Project API Key. This key serves two purposes:
1. It proves that the sender is authorized to write data to the Control Plane.
2. It identifies *which* project is sending the data, allowing the engine to apply the correct Dynamic Routing variables (Slack channels, Jira keys).

**How to authenticate:**
Pass your API Key in the `X-API-Key` HTTP Header.

```http
POST /api/webhooks/custom HTTP/1.1
Host: sre.yourcompany.com
Content-Type: application/json
X-API-Key: sre_pk_custom_9f8a7b6c5d4e3f2a1
```

*Note: API Keys are generated via the QA Capsule UI under CI/CD Gateways. If an API Key is compromised, simply delete the project endpoint in the UI and provision a new one.*

## 3. The EndpointPOST

`/api/webhooks/{provider}`

Ingests a new incident report into the QA Capsule database.

Path Parameters
* provider (string): A logical grouping for the source of the data.
  * Accepted values: `github`, `gitlab`, `jenkins`, `custom`.
  * Note: Using `custom` is recommended for internal scripts or unsupported CI providers.

## 4. The Unified Alert Schema (Request Body)

QA Capsule standardizes all errors into a single, predictable format called the Unified Alert Schema. The body of your POST request must be a valid JSON object adhering to this schema.

### JSON fields : 
| Field | Type  | Required  | Description  |
|---|---|---|---|
| `name`  | string  | Yes  | The title of the incident. Usually the name of the CI Job or the specific Test Case that failed. Max 255 chars.  |   
| `status`  | string  | Yes  | The severity of the alert. Must be exactly `CRITICAL`, `WARNING`, or `INFO`. (See Status Definitions below).  |   
| `error`  | string  | Yes  | A short, one-line summary of the failure (e.g., "AssertionError: Expected 200, got 500"). Max 500 chars.  |   
| `console_logs`  | string  | Yes  | The detailed technical payload. This should include the StackTrace, StdOut, and StdErr. Use `\n` for line breaks. This field is scanned by the Plugin Engine for trigger keywords (like `[FATAL]`).  |   
| `browser`  | string  | No  | Contextual metadata. Originally designed for E2E browser testing (e.g., Chrome 120), but can be used for any context (e.g., `Ubuntu 22.04`, `Node v18`).  |   

### Status Definitions & Plugin Triggers

* `CRITICAL`: Represents a hard failure (e.g., a broken build or a failed E2E test). This status actively triggers the Plugin Engine (sending Slack/Teams notifications and creating Jira tickets).

* `WARNING`: Represents a flaky test or a non-blocking error. It is logged in the Dashboard for visibility but does not trigger active notifications, preventing alert fatigue.

* `INFO`: Used for recording deployments or successful pipeline runs. Dropped by default unless specific plugins are configured to listen for it.

## 5. Example Requests
### Example 1: cURL

```json
  --url [https://sre.yourcompany.com/api/webhooks/custom](https://sre.yourcompany.com/api/webhooks/custom) \
  --header 'Content-Type: application/json' \
  --header 'X-API-Key: sre_pk_custom_a1b2c3d4' \
  --data '{
    "name": "Nightly Security Scan",
    "status": "CRITICAL",
    "error": "Vulnerability threshold exceeded: 3 High severity issues found.",
    "browser": "Docker/Trivy",
    "console_logs": "[FATAL] Scan failed.\n\n--- STACKTRACE ---\nCVE-2023-1234 found in package log4j.\nEnsure you update to version 2.17.1 immediately."
}'
```

### Example 2: Node.js (Fetch API)

If you are writing a custom reporter for a Node.js testing framework (like Jest or Mocha), you can push data natively:

```JavaScript
async function sendTelemetryToQACapsule(testResult) {
  const payload = {
    name: testResult.testName,
    status: testResult.isFlaky ? "WARNING" : "CRITICAL",
    error: testResult.failureMessage.substring(0, 200), // Keep it short
    browser: process.env.CI_ENVIRONMENT || "Local Dev",
    console_logs: `[FATAL] Test failed.\n\n--- STACKTRACE ---\n${testResult.stackTrace}`
  };

  try {
    const response = await fetch('[https://sre.yourcompany.com/api/webhooks/custom](https://sre.yourcompany.com/api/webhooks/custom)', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': process.env.QA_CAPSULE_API_KEY
      },
      body: JSON.stringify(payload)
    });

    if (response.status === 202) {
      console.log('Successfully pushed telemetry to QA Capsule.');
    } else {
      console.error(`QA Capsule API Error: ${response.status}`);
    }
  } catch (err) {
    console.error('Network error reaching QA Capsule', err);
  }
}
```

## 6. HTTP Responses

The API uses standard HTTP status codes to indicate the success or failure of an API 

### `request.202` 

AcceptedThe payload was successfully validated and queued for asynchronous processing.

```JSON
{
  "project": "Security Audits",
  "status": "incident_recorded",
  "timestamp": "2024-05-10T14:32:01Z"
}
```

### `400 Bad Request`

The JSON payload was malformed or missing required fields (name, status, error, console_logs).
```JSON
{
  "error": "validation_failed",
  "message": "Field 'status' must be CRITICAL, WARNING, or INFO."
}
```

### `401 Unauthorized`

Authentication failed. This happens if the X-API-Key header is completely missing.

```JSON
{
  "error": "unauthorized",
  "message": "Missing X-API-Key Header"
}
```

### `403 Forbidden`

An API Key was provided, but it does not match any provisioned project in the QA Capsule database. Ensure the key was not revoked or deleted in the UI.

```JSON
{
  "error": "forbidden",
  "message": "Invalid API Key. Project not found."
}
```

### `413 Payload Too Large`
The incoming JSON body exceeds the server's limit (default is usually 5MB). If your `console_logs` string contains massive memory dumps, truncate it before sending.

### `429 Too Many Requests`

(If Rate Limiting is enabled at your reverse proxy layer). You are sending too many requests per second. Implement an exponential backoff retry logic in your SRE Agent.