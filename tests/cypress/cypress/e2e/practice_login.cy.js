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
 * On failure the support file qa-capsule.js captures the full page HTML.
 * QA Capsule reads it from <system-out> in the uploaded JUnit XML.
 * The heal_incident MCP tool then gives the page HTML to the AI and asks
 * for the correct replacement locator.
 */

const BASE_URL = 'https://practicetestautomation.com/practice-test-login/';
const USERNAME  = 'student';
const PASSWORD  = 'Password123';

// -----------------------------------------------------------------
// BROKEN LOCATOR — intentionally wrong for the self-healing demo.
// The real submit button selector on this page is:  #submit
// The AI will read the captured DOM and propose the correct one.
// -----------------------------------------------------------------
const BROKEN_SUBMIT = '[aria-label="Sign in"]';

describe('Practice Login — QA Capsule Self-Healing Demo', () => {

  it('TC-01 Login page loads with correct heading', () => {
    cy.visit(BASE_URL);
    cy.title().should('match', /Practice Test Automation/i);
    cy.get('h2').should('contain.text', 'Test Login Page');
  });

  it('TC-02 Login form elements are visible', () => {
    cy.visit(BASE_URL);
    // All three selectors below are correct — this test always passes.
    cy.get('#username').should('be.visible');
    cy.get('#password').should('be.visible');
    cy.get('#submit').should('be.visible');
  });

  it('TC-03 Valid credentials log in successfully', () => {
    cy.visit(BASE_URL);
    cy.get('#username').type(USERNAME);
    cy.get('#password').type(PASSWORD);
    // Correct selector — this test always passes.
    cy.get('#submit').click();
    cy.get('h1').should('contain.text', 'Logged In Successfully');
  });

  it('TC-04 Submit with broken locator [self-healing demo]', () => {
    /*
     * This test FAILS on purpose.
     *
     * The selector BROKEN_SUBMIT does not exist on the page.
     * The page DOM (captured by qa-capsule.js on failure) shows the real element:
     *   <button class="btn btn-lg btn-primary btn-block" id="submit">Submit</button>
     *
     * When you open QA Capsule and call the heal_incident MCP tool,
     * the AI reads the captured HTML and suggests changing:
     *   BROKEN: [aria-label="Sign in"]
     *   FIXED:  #submit
     */
    cy.visit(BASE_URL);
    cy.get('#username').type(USERNAME);
    cy.get('#password').type(PASSWORD);
    // This will throw "element not found" — the selector is outdated.
    cy.get(BROKEN_SUBMIT).click();
    cy.get('h1').should('contain.text', 'Logged In Successfully');
  });

});
