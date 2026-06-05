/**
 * QA Capsule support file for Cypress.
 *
 * After every failed test this hook:
 *  1. Calls cy.document() to get the current page HTML.
 *  2. Wraps it in [QA_CAPSULE_DOM_SNAPSHOT_START] / [QA_CAPSULE_DOM_SNAPSHOT_END] markers.
 *  3. Logs it via cy.log() and task('log') so it appears in the JUnit
 *     <system-out> field that QA Capsule reads into console_logs.
 *
 * QA Capsule's heal_incident MCP tool extracts the DOM from console_logs,
 * passes it to Groq / Gemini, and finds the correct replacement locator.
 *
 * Reference this file in cypress.config.js:
 *   e2e: { supportFile: 'cypress/support/qa-capsule.js' }
 */

const MAX_HTML_CHARS = 24_000;

// Capture DOM on every test failure.
afterEach(function () {
  if (this.currentTest && this.currentTest.state === 'failed') {
    cy.document().then((doc) => {
      const html = doc.documentElement.outerHTML.slice(0, MAX_HTML_CHARS);
      const snapshot =
        `[QA_CAPSULE_DOM_SNAPSHOT_START]\n` +
        `${html}\n` +
        `[QA_CAPSULE_DOM_SNAPSHOT_END]`;
      // cy.log writes to the Cypress runner output which is captured in JUnit.
      cy.log(snapshot);
      // Also write to the Node process stdout for CI log capture.
      cy.task('qacLog', snapshot, { log: false });
    });
  }
});
