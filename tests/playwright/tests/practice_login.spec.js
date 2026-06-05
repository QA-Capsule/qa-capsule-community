/**
 * End-to-end login tests against practicetestautomation.com.
 *
 * Target: https://practicetestautomation.com/practice-test-login/
 * Known credentials: username=student  password=Password123
 *
 * Test structure (3 pass + 1 intentional broken-locator failure):
 *   TC-01  Page loads correctly          → passes
 *   TC-02  Login form elements visible   → passes
 *   TC-03  Valid credentials log in      → passes
 *   TC-04  Submit with broken locator    → FAILS (self-healing demo)
 *
 * On failure the fixture captures the page HTML.
 * The qa-capsule-reporter writes it to stderr as a DOM snapshot.
 * QA Capsule stores that in console_logs.
 * Running heal_incident via MCP then asks the AI to find the correct selector.
 */

// Use the QA Capsule fixture instead of the standard Playwright test object
// so DOM capture happens automatically on every failure.
const { test, expect } = require('../fixtures/qa-capsule-fixture');

const BASE_URL = 'https://practicetestautomation.com/practice-test-login/';
const USERNAME  = 'student';
const PASSWORD  = 'Password123';

// -----------------------------------------------------------------
// BROKEN LOCATOR — intentionally wrong for the self-healing demo.
// The real submit button selector on this page is:  #submit
// The AI will read the captured DOM and propose the correct one.
// -----------------------------------------------------------------
const BROKEN_SUBMIT = '[data-testid="login-button"]';

test.describe('Practice Login — QA Capsule Self-Healing Demo', () => {

  test('TC-01 Login page loads with correct heading', async ({ page }) => {
    await page.goto(BASE_URL);
    await expect(page).toHaveTitle(/Practice Test Automation/i);
    await expect(page.locator('h2')).toContainText('Test Login Page');
  });

  test('TC-02 Login form elements are visible', async ({ page }) => {
    await page.goto(BASE_URL);
    // All three selectors below are correct — this test always passes.
    await expect(page.locator('#username')).toBeVisible();
    await expect(page.locator('#password')).toBeVisible();
    await expect(page.locator('#submit')).toBeVisible();
  });

  test('TC-03 Valid credentials log in successfully', async ({ page }) => {
    await page.goto(BASE_URL);
    await page.fill('#username', USERNAME);
    await page.fill('#password', PASSWORD);
    // Correct selector — this test always passes.
    await page.click('#submit');
    await expect(page.locator('h1')).toContainText('Logged In Successfully');
  });

  test('TC-04 Submit with broken locator [self-healing demo]', async ({ page }) => {
    /*
     * This test FAILS on purpose.
     *
     * The selector BROKEN_SUBMIT does not exist on the page.
     * The page DOM (captured automatically on failure) shows the real element:
     *   <button class="btn btn-lg btn-primary btn-block" id="submit">Submit</button>
     *
     * When you open QA Capsule and call the heal_incident MCP tool,
     * the AI reads the captured HTML and suggests changing:
     *   BROKEN: [data-testid="login-button"]
     *   FIXED:  #submit
     */
    await page.goto(BASE_URL);
    await page.fill('#username', USERNAME);
    await page.fill('#password', PASSWORD);
    // This will throw "locator not found" — the selector is outdated.
    await page.locator(BROKEN_SUBMIT).click({ timeout: 5000 });
    await expect(page.locator('h1')).toContainText('Logged In Successfully');
  });

});
