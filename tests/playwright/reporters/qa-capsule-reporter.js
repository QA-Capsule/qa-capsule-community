/**
 * QA Capsule DOM Capture Reporter for Playwright.
 *
 * This reporter reads test attachments that were written by the
 * qa-capsule-fixture.js afterEach hook.  When a test fails and the page
 * content was captured, this reporter writes the DOM snapshot (wrapped in
 * QA Capsule markers) to process.stderr.
 *
 * Playwright's JUnit reporter always includes stderr content in the
 * <system-err> element of each <testcase>, which QA Capsule ingests into
 * the console_logs column.  The heal_incident MCP tool then uses that HTML
 * to ask the AI for the correct replacement locator.
 *
 * Nothing in this file accesses the browser — it only reads attachments
 * that the fixture layer already captured.
 */

const MAX_HTML_CHARS = 24_000;

class QACapsuleReporter {
  onTestEnd(test, result) {
    if (result.status === 'passed' || result.status === 'skipped') return;

    // Look for a DOM snapshot attachment written by the fixture.
    const domAttachment = result.attachments.find(
      (a) => a.name === 'qa-capsule-dom-snapshot'
    );
    if (!domAttachment || !domAttachment.body) return;

    const html = domAttachment.body.toString('utf8').slice(0, MAX_HTML_CHARS);
    const snapshot =
      `\n[QA_CAPSULE_DOM_SNAPSHOT_START]\n` +
      `${html}\n` +
      `[QA_CAPSULE_DOM_SNAPSHOT_END]\n`;

    // stderr is captured by Playwright's JUnit reporter into <system-err>.
    process.stderr.write(snapshot);
  }

  onEnd() {}
}

module.exports = QACapsuleReporter;
