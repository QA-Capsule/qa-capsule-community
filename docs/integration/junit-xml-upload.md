---
icon: material/xml
---

# JUnit XML Upload API

The **JUnit XML Upload** endpoint is the recommended way to send test results to QA Capsule. It parses standard JUnit XML reports and creates one **sub-alert per failed test case**.

---

## Endpoint

```
POST /api/webhooks/upload?framework={FrameworkName}
```

| Parameter | Location | Required | Description |
|---|---|---|---|
| `framework` | Query string | Recommended | `Playwright`, `Cypress`, `Pytest`, `JUnit`, `RobotFramework`, `Postman`, etc. |
| `X-API-Key` | HTTP Header | **Yes** | Project API key from CI/CD Gateways |
| `X-Run-Id` | HTTP Header | Recommended | Groups results in Telemetry Stream (`github.run_id`, `$CI_PIPELINE_ID`, …) |
| `X-Execution-Env` | HTTP Header | Optional | `PROD`, `STAGING`, `DEV`, `CANARY` |
| `X-Execution-Type` | HTTP Header | Optional | `TEST-RUN`, `SMOKE`, `NIGHTLY`, `REAL` |
| `file` | Multipart body | **Yes** | JUnit XML file (max 10 MB) |

---

## Authentication

```http
POST /api/webhooks/upload?framework=Playwright HTTP/1.1
Host: sre.yourcompany.com
X-API-Key: sre_pk_github_a1b2c3d4e5f6
Content-Type: multipart/form-data; boundary=----boundary
```

No JWT is required for this endpoint — only the project API key.

---

## What Gets Parsed

For each `<testcase>` that contains a `<failure>` or `<error>` child, QA Capsule extracts:

| Field | XML source | Stored as |
|---|---|---|
| Test name | `testcase@name` + `testcase@class` | `incidents.name` |
| Error summary | `failure@message` or `failure` text | `incidents.error_message` |
| Standard output | `<system-out>` | `incidents.console_logs` |
| Standard error | `<system-err>` | `incidents.error_logs` |
| Fingerprint | SHA-256(`name` + `error`) | `incidents.fingerprint` |

The full report stores **all** test cases (pass, fail, skip) for the unified matrix view. Failed `<testcase>` nodes also create **incidents** (unless quarantined or deduplicated by fingerprint).

!!! note "Robot Framework nested suites"
    rebot xUnit may nest `<testsuite>` elements. QA Capsule flattens nested suites when building the execution report. Use `rebot --xunit robot-junit.xml --outputdir tests/results` (basename only for `--xunit`).

See [All test frameworks](test-frameworks.md) (Playwright, Cypress, Pytest, Robot, Newman, JUnit, …).

---

## Example XML Input

```xml
<?xml version="1.0" encoding="UTF-8"?>
<testsuites>
  <testsuite name="Checkout Suite" tests="2" failures="1">
    <testcase classname="checkout.spec" name="payment button visible">
      <failure message="Timeout 1500ms exceeded" type="TimeoutError">
        locator.click: Timeout 1500ms exceeded.
        Call log:
          - waiting for locator('#stripe-pay-button')
      </failure>
      <system-out>Navigating to checkout...</system-out>
      <system-err>Payment gateway slow response</system-err>
    </testcase>
    <testcase classname="checkout.spec" name="cart total correct"/>
  </testsuite>
</testsuites>
```

This creates **one incident** for `payment button visible` and skips the passing test.

---

## cURL Example

```bash
curl -f -X POST "https://sre.yourcompany.com/api/webhooks/upload?framework=Playwright" \
  -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
  -F "file=@playwright-results.xml"
```

---

## GitHub Actions Example

```yaml
- name: Run Playwright Tests
  run: npx playwright test
  continue-on-error: true

- name: Upload results to QA Capsule
  if: always()
  env:
    QA_CAPSULE_URL: ${{ secrets.QA_CAPSULE_URL }}
    QA_CAPSULE_API_KEY: ${{ secrets.QA_CAPSULE_API_KEY }}
  run: |
    curl -f -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=Playwright" \
      -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
      -F "file=@playwright-results.xml"
```

---

## GitLab CI Example

```yaml
e2e_tests:
  script:
    - npx playwright test
  after_script:
    - |
      curl -f -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=Playwright" \
        -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
        -F "file=@playwright-results.xml"
  variables:
    QA_CAPSULE_URL: $QA_CAPSULE_URL
    QA_CAPSULE_API_KEY: $QA_CAPSULE_API_KEY
```

---

## HTTP Responses

### `202 Accepted` — Failures processed

```json
{
  "status": "success",
  "failures_processed": 3,
  "project": "Frontend E2E - Playwright"
}
```

### `200 OK` — No tests parsed / no failures

```json
{
  "status": "success",
  "message": "No actionable events detected."
}
```

This usually means the XML was empty, malformed, or no `<testcase>` nodes were found (e.g. wrong file path, or nested Robot suites on an older server). Check CI logs and [Test frameworks — troubleshooting](test-frameworks.md#troubleshooting).

### `401 Unauthorized`

Missing or invalid `X-API-Key` header.

### `400 Bad Request`

- Multipart form missing `file` field
- File exceeds 10 MB limit
- Malformed XML

---

## Correlation Rules (Post-Upload)

After parsing, each failure goes through correlation:

1. **Open incident with same fingerprint exists** → skip (log: `already open`).
2. **Resolved incident with same fingerprint exists** → skip (manual resolution preserved).
3. **Previously resolved within 48h** → new incident tagged `[FLAKY] TestName`.
4. **Otherwise** → new open incident created, plugins triggered if `CRITICAL`.

---

## Best Practices

1. **Always use `if: always()`** so the upload runs when tests fail.
2. **Use `-f` on curl** so CI fails visibly if QA Capsule is unreachable.
3. **One XML file per job** — merge reports if you have sharded test runs.
4. **Keep XML on disk** as a CI artifact for debugging alongside QA Capsule incidents.
5. **Match `framework` query param** to your actual test runner for easier filtering in logs.
