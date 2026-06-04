/**
 * QA Capsule — Playwright JS/TS DOM Capture Reporter
 * Captures page HTML + screenshot on every test failure
 * and prints it in the format QA Capsule expects for AI locator healing.
 *
 * Usage in playwright.config.ts:
 *   reporter: [['list'], ['./tests/dom-capture/playwright-js/qa-capsule-reporter.js']]
 */

const MAX_HTML = 24_000;

class QACapsuleReporter {
  onTestEnd(test, result) {
    if (result.status === 'passed') return;

    for (const attachment of result.attachments) {
      if (attachment.name === 'qa-capsule-dom' && attachment.body) {
        const html = attachment.body.toString('utf-8').slice(0, MAX_HTML);
        process.stdout.write(
          `\n[QA_CAPSULE_DOM_SNAPSHOT_START]\n${html}\n[QA_CAPSULE_DOM_SNAPSHOT_END]\n`
        );
      }
    }
  }
}

module.exports = QACapsuleReporter;
