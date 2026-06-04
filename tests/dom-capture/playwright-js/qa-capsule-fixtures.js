/**
 * QA Capsule — Playwright JS/TS DOM Capture Fixture
 * Wraps every test with automatic DOM capture on failure.
 *
 * Usage:
 *   // In your test file, import from this fixture instead of @playwright/test
 *   import { test, expect } from '../dom-capture/playwright-js/qa-capsule-fixtures';
 *
 *   test('my test', async ({ page }) => { ... });
 */

const { test: base } = require('@playwright/test');

const test = base.extend({
  page: async ({ page }, use, testInfo) => {
    await use(page);

    if (testInfo.status !== 'passed') {
      try {
        const html = await page.content();
        await testInfo.attach('qa-capsule-dom', {
          body: Buffer.from(html, 'utf-8'),
          contentType: 'text/html',
        });
        const screenshot = await page.screenshot({ fullPage: true });
        await testInfo.attach('qa-capsule-screenshot', {
          body: screenshot,
          contentType: 'image/png',
        });
      } catch (_) {}
    }
  },
});

module.exports = { test, expect: base.expect };
