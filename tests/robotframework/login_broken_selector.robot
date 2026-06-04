*** Settings ***
Documentation
...    Simulated login flow with a BROKEN CSS selector.
...    This test WILL FAIL on purpose — the selector "button[data-qa='submit-v1']"
...    no longer exists after the UI redesign (v2 uses data-qa='submit-login').
...    QA Capsule will capture the failure and Gemini will suggest the fix.
Resource         resources/common.robot
Library          Browser
Suite Setup      Log Suite Context

*** Variables ***
${LOGIN_URL}        https://practicetestautomation.com/practice-test-login/
${USERNAME}         student
${PASSWORD}         Password123
# intentionally wrong — the real button id is "submit"
${SUBMIT_BTN}       button[data-qa="submit-v1"]

*** Test Cases ***

User Can Log In With Valid Credentials
    [Documentation]
    ...    Opens the login page and clicks the submit button.
    ...    FAILS because ${SUBMIT_BTN} does not exist on the page.
    [Tags]    login    broken-selector    self-healing-demo
    New Browser    chromium    headless=true
    New Page    ${LOGIN_URL}
    Fill Text    id=username    ${USERNAME}
    Fill Text    id=password    ${PASSWORD}
    # This line will raise ElementNotFoundError — the selector is outdated
    Click    ${SUBMIT_BTN}
    Wait For Elements State    h1    visible    timeout=5s
    Get Text    h1    ==    Logged In Successfully
    [Teardown]    Close Browser

Password Field Rejects Empty Input
    [Documentation]
    ...    Verifies the error message when submitting an empty password.
    ...    Also uses the broken selector — same root cause.
    [Tags]    login    broken-selector    self-healing-demo
    New Browser    chromium    headless=true
    New Page    ${LOGIN_URL}
    Fill Text    id=username    ${USERNAME}
    Fill Text    id=password    ${EMPTY}
    Click    ${SUBMIT_BTN}
    Wait For Elements State    id=error    visible    timeout=5s
    Get Text    id=error    contains    Your password is invalid
    [Teardown]    Close Browser
