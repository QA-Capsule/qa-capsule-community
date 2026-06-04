/**
 * QA Capsule — Cypress DOM Capture Plugin
 * Captures page HTML on every test failure for AI locator healing.
 *
 * Installation:
 *   1. Copy this file to cypress/support/qa-capsule-plugin.js
 *   2. In cypress/support/e2e.js, add:
 *        import './qa-capsule-plugin';
 *   3. In cypress.config.js, add the task in setupNodeEvents:
 *        require('./cypress/support/qa-capsule-plugin').registerTasks(on)
 */

const MAX_HTML = 24_000;

// ── Browser-side: capture DOM after each failed test ─────────────────────────
afterEach(function () {
  if (this.currentTest?.state !== 'failed') return;

  cy.document().then((doc) => {
    const html = doc.documentElement.outerHTML.slice(0, MAX_HTML);
    cy.task('qaCapsuleDomCapture', { html, testTitle: this.currentTest.title }, { log: false });
  });

  cy.screenshot(`qa_capsule_failure_${Date.now()}`, { capture: 'fullPage' });
});

// ── Node-side: print DOM snapshot to stdout for JUnit capture ─────────────────
function registerTasks(on) {
  on('task', {
    qaCapsuleDomCapture({ html, testTitle }) {
      process.stdout.write(
        `\n[QA_CAPSULE_DOM_SNAPSHOT_START]\n${html}\n[QA_CAPSULE_DOM_SNAPSHOT_END]\n`
      );
      return null;
    },
  });
}

module.exports = { registerTasks };
