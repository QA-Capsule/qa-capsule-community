/**
 * Cypress configuration for the QA Capsule self-healing demo.
 *
 * The support file (cypress/support/qa-capsule.js) hooks into afterEach to
 * capture the page HTML on every test failure.  It wraps the HTML in
 * QA_CAPSULE_DOM_SNAPSHOT_START / END markers and writes them to cy.log and
 * to Node stdout via a cy.task.  mocha-junit-reporter then includes the log
 * content in the <system-out> field of each failing <testcase> in the JUnit
 * XML, which is what QA Capsule reads as console_logs during ingestion.
 */
const { defineConfig } = require('cypress');

module.exports = defineConfig({
  e2e: {
    supportFile: 'cypress/support/qa-capsule.js',
    // Required to navigate to cross-origin test sites.
    chromeWebSecurity: false,
    setupNodeEvents(on) {
      // Simple task that writes to Node stdout so CI logs capture the snapshot.
      on('task', {
        qacLog(message) {
          process.stdout.write(message + '\n');
          return null;
        },
      });
    },
  },
  reporter: 'mocha-junit-reporter',
  reporterOptions: {
    mochaFile: 'cypress-results.xml',
    toConsole: true,
    includePending: true,
  },
  defaultCommandTimeout: 10000,
  pageLoadTimeout: 30000,
  video: false,
  screenshotOnRunFailure: false,
});
