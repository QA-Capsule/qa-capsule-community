---
icon: material/microsoft-edge
---

# Playwright

| | |
|---|---|
| **Upload param** | `?framework=Playwright` |
| **Report** | `playwright-results.xml` |
| **Repo workflow** | `.github/workflows/e2e-tests-playwright.yml` |

## 1. JUnit reporter

```javascript
// playwright.config.js
import { defineConfig } from '@playwright/test';

export default defineConfig({
  reporter: [
    ['list'],
    ['junit', { outputFile: 'playwright-results.xml' }],
  ],
});
```

## 2. Run tests

```bash
npm ci
npx playwright install --with-deps chromium
npx playwright test
```

## 3. Upload to QA Capsule

```bash
curl -f -S -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=Playwright" \
  -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
  -H "X-Run-Id: ${GITHUB_RUN_ID}" \
  -F "file=@playwright-results.xml"
```

## 4. GitHub Actions

```yaml
- run: npx playwright test
  continue-on-error: true

- name: Upload to QA Capsule
  if: always()
  env:
    QA_CAPSULE_URL: ${{ secrets.QA_CAPSULE_URL }}
    QA_CAPSULE_API_KEY: ${{ secrets.QA_CAPSULE_API_PLAYWRIGHT_KEY }}
  run: |
    curl -f -S -X POST "${QA_CAPSULE_URL}/api/webhooks/upload?framework=Playwright" \
      -H "X-API-Key: ${QA_CAPSULE_API_KEY}" \
      -H "X-Run-Id: ${{ github.run_id }}" \
      -F "file=@playwright-results.xml"
```

## Alternative

Real-time reporter + traces: [Playwright Reporter](../playwright-reporter.md)

← [All frameworks](../test-frameworks.md)
