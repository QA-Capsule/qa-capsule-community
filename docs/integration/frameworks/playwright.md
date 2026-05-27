---
icon: material/microsoft-edge
---

# Playwright

| | |
|---|---|
| **Upload param** | `?framework=Playwright` |
| **Report** | `playwright-results.xml` |
| **Repo workflow** | `.github/workflows/e2e-tests-playwright.yml` |
| **Secret** | `QA_CAPSULE_API_PLAYWRIGHT_KEY` |

## Test suites in this repository

| Suite | Tests | Expected result |
|---|---|---|
| `Navigation & Layout` | homepage title, mobile hamburger menu, 404 status | 1 pass · 2 fail |
| `Form Interactions` | contact form submit, empty form rejection, password strength | 1 pass · 2 fail |
| `API & Network Layer` | GET health 200, POST orders 503, response schema | 2 pass · 1 fail |

## 1. Install Playwright

```bash
npm install -D @playwright/test
npx playwright install --with-deps chromium
```

## 2. Configure JUnit reporter

```javascript
// playwright.config.js
module.exports = {
  reporter: [['junit', { outputFile: 'playwright-results.xml' }]],
};
```

## 3. Run tests

```bash
npx playwright test
```

## 4. Upload to QA Capsule

```bash
curl -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=Playwright" \
  -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
  -H "X-Run-Id: ${GITHUB_RUN_ID}" \
  -H "X-Execution-Env: STAGING" \
  -H "X-Execution-Type: TEST-RUN" \
  -F "file=@playwright-results.xml"
```

## 5. GitHub Actions

```yaml
- name: Run Playwright Tests
  run: npx playwright test suite.spec.js
  continue-on-error: true

- name: Send Alert to QA Capsule
  if: always()
  env:
    WEBHOOK_URL: ${{ secrets.QA_CAPSULE_URL }}
    API_KEY: ${{ secrets.QA_CAPSULE_API_PLAYWRIGHT_KEY }}
  run: |
    curl -X POST "$WEBHOOK_URL/api/webhooks/upload?framework=Playwright" \
      -H "X-API-Key: $API_KEY" \
      -H "X-Run-Id: ${{ github.run_id }}" \
      -H "X-Execution-Env: STAGING" \
      -H "X-Execution-Type: TEST-RUN" \
      -F "file=@playwright-results.xml"
```

!!! note "Headers"
    - `X-Run-Id` groups all test results under the same pipeline run in the Execution Hub.
    - `X-Execution-Env` accepts `PROD`, `STAGING`, `INTEGRATION`, `DEV`.
    - `X-Execution-Type` accepts `TEST-RUN`, `NIGHTLY`, `SMOKE`, `REAL`.

## Alternative: real-time reporter with traces

For step-level logs and Playwright trace attachments, use the native reporter:
[Playwright Reporter](../playwright-reporter.md)

← [All frameworks](../test-frameworks.md)
