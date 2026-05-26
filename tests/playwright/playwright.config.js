// Canonical Playwright sample for QA Capsule upload.
module.exports = {
  reporter: [
    ["list"],
    ["junit", { outputFile: "playwright-results.xml" }]
  ],
  use: {
    headless: true
  }
};
