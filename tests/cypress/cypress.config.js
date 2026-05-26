const { defineConfig } = require("cypress");

module.exports = defineConfig({
  e2e: { supportFile: false },
  reporter: "mocha-junit-reporter",
  reporterOptions: {
    mochaFile: "cypress-results.xml",
    toConsole: true
  }
});
