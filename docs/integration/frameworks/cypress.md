---
icon: material/web
---

# Cypress

| | |
|---|---|
| **Upload param** | `?framework=Cypress` |
| **Report** | `cypress-results.xml` (or `cypress/results/*.xml`) |
| **Repo workflow** | `.github/workflows/e2e-tests-cypress.yml` |

## 1. JUnit reporter

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

```bash
npm install -D cypress mocha-junit-reporter
```

## 2. Run tests

```bash
npx cypress run
```

## 3. Upload

```bash
curl -f -S -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=Cypress" \
  -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
  -H "X-Run-Id: ${CI_PIPELINE_ID}" \
  -F "file=@cypress-results.xml"
```

## 4. GitHub Actions

```yaml
- run: npx cypress run
  continue-on-error: true

- name: Upload to QA Capsule
  if: always()
  env:
    QA_CAPSULE_URL: ${{ secrets.QA_CAPSULE_URL }}
    QA_CAPSULE_API_KEY: ${{ secrets.QA_CAPSULE_API_CYPRESS_KEY }}
  run: |
    curl -f -S -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=Cypress" \
      -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
      -H "X-Run-Id: ${{ github.run_id }}" \
      -F "file=@cypress-results.xml"
```

← [All frameworks](../test-frameworks.md)
