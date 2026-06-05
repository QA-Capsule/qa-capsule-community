*** Settings ***
Documentation
...    End-to-end login tests against practicetestautomation.com.
...
...    The site is a permanent test-automation sandbox.  Known credentials:
...      username : student
...      password : Password123
...
...    Test structure (3 pass + 1 intentional broken-locator failure):
...      1. Page loads correctly          → passes
...      2. Login form elements present   → passes
...      3. Successful login flow         → passes
...      4. Submit with broken locator    → FAILS on purpose (self-healing demo)
...
...    On failure the dom_capture_listener captures the live page HTML.
...    QA Capsule stores that HTML in console_logs.  When you call the MCP
...    tool heal_incident the AI reads the HTML, finds the real selector,
...    and proposes the fix automatically.

Resource    resources/common.robot
Library     Browser

Suite Setup      New Browser    chromium    headless=true
Suite Teardown   Close Browser

*** Variables ***
${BASE_URL}         https://practicetestautomation.com/practice-test-login/
${VALID_USER}       student
${VALID_PASS}       Password123
# -----------------------------------------------------------
# BROKEN LOCATOR — this is intentionally wrong.
# The real submit button selector is:  #submit
# After healing, QA Capsule AI will suggest the correction.
# -----------------------------------------------------------
${BROKEN_SUBMIT}    button[data-qa="submit-v1"]

*** Test Cases ***

TC-01 Login Page Loads Successfully
    [Documentation]
    ...    Navigates to the login page and verifies the title and heading.
    ...    This test always passes — it validates the environment is reachable.
    [Tags]    smoke    login    passing
    New Page    ${BASE_URL}
    Get Title    contains    Practice Test Automation
    Get Text     h1    contains    Test Login Page
    Close Page

TC-02 Login Form Elements Are Present
    [Documentation]
    ...    Checks that username field, password field and submit button exist.
    ...    Uses stable, correct selectors — always passes.
    [Tags]    smoke    login    passing
    New Page    ${BASE_URL}
    Get Element    id=username
    Get Element    id=password
    Get Element    id=submit
    Close Page

TC-03 Valid Credentials Log In Successfully
    [Documentation]
    ...    Fills the form with correct credentials and confirms the success page.
    ...    Uses correct selectors — always passes.
    [Tags]    login    passing    functional
    New Page    ${BASE_URL}
    Fill Text    id=username    ${VALID_USER}
    Fill Text    id=password    ${VALID_PASS}
    Click        id=submit
    Get Text     h1    contains    Logged In Successfully
    Close Page

TC-04 Submit Button With Broken Locator (Self-Healing Demo)
    [Documentation]
    ...    This test FAILS intentionally to demonstrate QA Capsule self-healing.
    ...
    ...    The selector "${BROKEN_SUBMIT}" does not exist on the page.
    ...    The real selector is "#submit".
    ...
    ...    When this test fails, dom_capture_listener captures the full page HTML.
    ...    QA Capsule stores it in console_logs.  The MCP tool heal_incident
    ...    then calls the AI with the page HTML to find the correct selector.
    [Tags]    login    broken-locator    self-healing-demo
    New Page    ${BASE_URL}
    Fill Text    id=username    ${VALID_USER}
    Fill Text    id=password    ${VALID_PASS}
    # This click will raise ElementNotFoundError — selector is outdated
    Click        ${BROKEN_SUBMIT}
    Get Text     h1    contains    Logged In Successfully
    Close Page
