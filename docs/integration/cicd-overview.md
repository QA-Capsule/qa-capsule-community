---
icon: material/sitemap-outline
---

# CI/CD Integration — Complete Guide

This page is the **master reference** for connecting any CI/CD platform to QA Capsule. Read this first, then follow the provider-specific guide for your stack.

---

## Two Ingestion Methods

QA Capsule supports two complementary ingestion paths:

| Method | Endpoint | Best for |
|---|---|---|
| **JUnit XML Upload** (recommended) | `POST /api/webhooks/upload` | Playwright, Cypress, Pytest, JUnit, Robot Framework — any framework that exports XML |
| **JSON Webhook** | `POST /api/webhooks/{provider}` | Custom scripts, lightweight notifications, legacy integrations |

!!! tip "Always prefer JUnit XML when possible"
    The upload endpoint parses each failed `<testcase>` individually, creates one sub-alert per test, captures `system-out` / `system-err`, and enables accurate fingerprinting. JSON webhooks are better for pipeline-level alerts without structured test reports.

---

## Integration Checklist

Use this checklist for **every** new project:

- [ ] **1. Provision a project** in QA Capsule → **CI/CD Gateways** (Ingestion module).
- [ ] **2. Copy** the generated **Webhook URL** and **API Key** immediately (the key is shown only once).
- [ ] **3. Store secrets** in your CI platform (GitHub Secrets, GitLab CI Variables, Jenkins Credentials).
- [ ] **4. Configure your test framework** to output JUnit XML (see [JUnit XML Upload](junit-xml-upload.md)).
- [ ] **5. Add a post-test step** with `if: always()` (GitHub) or `after_script` (GitLab) so telemetry is sent even when tests fail.
- [ ] **6. Upload the XML** via `curl` multipart form to `/api/webhooks/upload?framework={Name}`.
- [ ] **7. Verify** the incident appears in the Dashboard within seconds.
- [ ] **8. Configure routing** (Slack channel, Teams webhook, Jira key) in the project provisioning form.
- [ ] **9. Activate plugins** under the Plugins module if you want automatic notifications.

---

## Required CI/CD Secrets

Regardless of platform, you need exactly two secrets:

| Secret name (example) | Value |
|---|---|
| `QA_CAPSULE_URL` | Base URL of your instance, e.g. `https://sre.yourcompany.com` — **no trailing slash** |
| `QA_CAPSULE_API_KEY` | Project API key from the provisioning screen, e.g. `sre_pk_github_abc123` |

The upload URL is always:

```
${QA_CAPSULE_URL}/api/webhooks/upload?framework=Playwright
```

---

## Universal Upload Step (All Platforms)

This `curl` command works on GitHub Actions, GitLab CI, Jenkins, Azure DevOps, CircleCI, and any Linux runner:

```bash
curl -f -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=Playwright" \
  -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
  -F "file=@test-results.xml"
```

| Flag | Purpose |
|---|---|
| `-f` | Fail the CI step if the upload returns HTTP 4xx/5xx (recommended) |
| `X-API-Key` | Authenticates and routes to the correct project |
| `framework` query param | Used for log context and parser hints (`Playwright`, `Cypress`, `Pytest`, `JUnit`) |
| `file` | Path to your JUnit XML report on the runner filesystem |

**Expected response:** `202 Accepted`

```json
{
  "status": "success",
  "failures_processed": 3,
  "project": "Frontend E2E - Playwright"
}
```

---

## Provider-Specific Guides

| Platform | Guide |
|---|---|
| GitHub Actions | [GitHub Actions Integration](github.md) |
| GitLab CI | [GitLab CI Integration](gitlab.md) |
| Jenkins | [Jenkins Integration](jenkins.md) |
| Custom / Internal CI | [Webhooks API](../api/webhooks.md) + [JUnit XML Upload](junit-xml-upload.md) |
| Shell agent script | [The SRE Agent](agent.md) |

---

## How Correlation Protects Your Dashboard

After you resolve an incident, QA Capsule **will not create a duplicate** if the same test fails again with the same fingerprint:

1. **Open duplicate** — same fingerprint + `is_resolved = 0` → skipped (no spam).
2. **Resolved duplicate** — same fingerprint + `is_resolved = 1` → skipped (your manual resolution is respected).
3. **New failure** — different error message → new fingerprint → new incident (legitimate new failure).

See [Incident Lifecycle](../guides/incident-lifecycle.md) for full details.

---

## Framework Configuration Examples

### Playwright

```javascript
// playwright.config.js
module.exports = {
  reporter: [['junit', { outputFile: 'playwright-results.xml' }]],
};
```

### Cypress

```javascript
// cypress.config.js
module.exports = {
  reporter: 'junit',
  reporterOptions: {
    mochaFile: 'cypress/results/test-results.xml',
  },
};
```

### Pytest

```bash
pytest --junitxml=pytest-results.xml
```

### Maven / Java

```xml
<plugin>
  <groupId>org.apache.maven.plugins</groupId>
  <artifactId>maven-surefire-plugin</artifactId>
  <configuration>
    <reportsDirectory>${project.build.directory}/surefire-reports</reportsDirectory>
  </configuration>
</plugin>
```

---

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `401 Unauthorized` | Missing or wrong `X-API-Key` | Re-copy key from CI/CD Gateways; check for trailing spaces in secrets |
| `404 Not Found` | Wrong URL path | Use `/api/webhooks/upload`, not `/api/webhook/upload` |
| No incidents on dashboard | Upload step skipped on failure | Add `if: always()` (GitHub) or `after_script` (GitLab) |
| Incidents revert to active after resolve | Old behavior — fixed in current version | Upgrade to latest; resolved fingerprints are now suppressed |
| Duplicate alerts every 5 min | CI re-sends same failure | Resolve the incident; correlation will suppress re-ingestion |
| Empty XML / 200 with 0 failures | All tests passed | Expected — no incidents created when XML has no failures |
