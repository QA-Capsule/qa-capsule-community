---
icon: material/web
---

# Cypress

| | |
|---|---|
| **Upload param** | `?framework=Cypress` |
| **Report** | `cypress-results.xml` |
| **Repo workflow** | `.github/workflows/e2e-tests-cypress.yml` |
| **Secret** | `QA_CAPSULE_API_CYPRESS_KEY` |

## Test suites in this repository

| Suite | Tests | Expected result |
|---|---|---|
| `Authentication & Session Tests` | login page elements, missing password, dashboard redirect | 2 pass · 1 fail |
| `Product Catalog Tests` | product grid, category filter, add to cart | 1 pass · 2 fail |
| `Checkout Flow Tests` | order total, invalid card, order confirmation | 2 pass · 1 fail |

## 1. Install JUnit reporter

```bash
npm install -D cypress mocha-junit-reporter
```

## 2. Configure Cypress

```javascript
// cypress.config.js
const { defineConfig } = require('cypress');

module.exports = defineConfig({
  e2e: { supportFile: false },
  reporter: 'mocha-junit-reporter',
  reporterOptions: {
    mochaFile: 'cypress-results.xml',
    toConsole: true,
  },
});
```

## 3. Run tests

```bash
npx cypress run
```

## 4. Upload to QA Capsule

```bash
curl -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=Cypress" \
  -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
  -H "X-Run-Id: ${CI_PIPELINE_ID}" \
  -H "X-Execution-Env: STAGING" \
  -H "X-Execution-Type: TEST-RUN" \
  -F "file=@cypress-results.xml"
```

## 5. GitHub Actions

```yaml
- name: Run Cypress Tests
  run: npx cypress run
  continue-on-error: true

- name: Send Alert to QA Capsule
  if: always()
  env:
    WEBHOOK_URL: ${{ secrets.QA_CAPSULE_URL }}
    API_KEY: ${{ secrets.QA_CAPSULE_API_CYPRESS_KEY }}
  run: |
    curl -X POST "$WEBHOOK_URL/api/webhooks/upload?framework=Cypress" \
      -H "X-API-Key: $API_KEY" \
      -H "X-Run-Id: ${{ github.run_id }}" \
      -H "X-Execution-Env: STAGING" \
      -H "X-Execution-Type: TEST-RUN" \
      -F "file=@cypress-results.xml"
```

!!! note "Headers"
    - `X-Run-Id` groups all test results under the same pipeline run in the Execution Hub.
    - `X-Execution-Env` accepts `PROD`, `STAGING`, `INTEGRATION`, `DEV`.
    - `X-Execution-Type` accepts `TEST-RUN`, `NIGHTLY`, `SMOKE`, `REAL`.

← [All frameworks](../test-frameworks.md)
