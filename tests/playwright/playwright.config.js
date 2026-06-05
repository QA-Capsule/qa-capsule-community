// Playwright configuration for the QA Capsule self-healing demo suite.
//
// The qa-capsule reporter (./reporters/qa-capsule-reporter.js) is included
// alongside the standard JUnit reporter.  On every test failure it captures
// the full page HTML, wraps it in QA_CAPSULE_DOM_SNAPSHOT_START/END markers,
// and writes it to stderr — which Playwright's JUnit reporter includes in
// <system-err> of the failing <testcase>.  QA Capsule reads that field into
// console_logs so the AI heal_incident tool has the DOM context it needs.

const { defineConfig, devices } = require('@playwright/test');

module.exports = defineConfig({
  testDir: './tests',
  timeout: 30_000,
  retries: 0,
  workers: 1,
  reporter: [
    ['list'],
    ['junit', { outputFile: 'playwright-results.xml' }],
    // Custom reporter that attaches the DOM snapshot to each failing test.
    ['./reporters/qa-capsule-reporter.js'],
  ],
  use: {
    headless: true,
    screenshot: 'only-on-failure',
    video: 'off',
    trace: 'off',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
