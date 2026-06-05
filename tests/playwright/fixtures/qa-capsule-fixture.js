/**
 * QA Capsule test fixture for Playwright.
 *
 * Extends the standard Playwright test object with an afterEach hook that
 * captures the full page HTML whenever a test fails.  The HTML is attached
 * to the test result under the name "qa-capsule-dom-snapshot".
 *
 * The qa-capsule-reporter.js then picks up that attachment and writes it to
 * stderr so it ends up in the <system-err> field of the JUnit XML that QA
 * Capsule ingests.
 *
 * Usage in test files — replace the standard import:
 *
 *   // Before (standard Playwright):
 *   const { test, expect } = require('@playwright/test');
 *
 *   // After (with QA Capsule DOM capture):
 *   const { test, expect } = require('../fixtures/qa-capsule-fixture');
 *
 * No other changes needed in the test file.
 */

const { test: baseTest, expect } = require('@playwright/test');

const test = baseTest.extend({
  // Override the page fixture to add automatic DOM capture on failure.
  page: async ({ page }, use, testInfo) => {
    // Run the test normally.
    await use(page);

    // After the test: capture DOM if the test failed.
    if (testInfo.status !== testInfo.expectedStatus) {
      try {
        const html = await page.content();
        if (html) {
          await testInfo.attach('qa-capsule-dom-snapshot', {
            body: Buffer.from(html, 'utf8'),
            contentType: 'text/html',
          });
        }
      } catch (_) {
        // Page may be closed already — capture is best-effort.
      }
    }
  },
});

module.exports = { test, expect };
