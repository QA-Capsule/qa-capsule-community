*** Settings ***
Documentation    UI navigation template (Selenium). Disabled in CI unless SELENIUM_ENABLED=true.
Resource         resources/common.robot
Library          SeleniumLibrary
Suite Setup      Log Suite Context
Suite Teardown    Run Keyword And Ignore Error    Close All Browsers

*** Variables ***
${SELENIUM_ENABLED}    %{SELENIUM_ENABLED=false}
${BROWSER}             %{SELENIUM_BROWSER=headlesschrome}
${BASE_URL}            %{UI_BASE_URL=https://example.com}

*** Test Cases ***
Example Domain Title Is Correct
    [Documentation]    Opens example.com and checks the page title.
    [Tags]    ui    selenium
    Skip If Selenium Disabled
    Open Browser    ${BASE_URL}    ${BROWSER}
    Title Should Be    Example Domain
    Page Should Contain    Example Domain

Navigation Link Is Present
    [Documentation]    Verifies a stable element exists on the landing page.
    [Tags]    ui    selenium
    Skip If Selenium Disabled
    Open Browser    ${BASE_URL}    ${BROWSER}
    Page Should Contain Element    css:a[href*="iana.org"]

*** Keywords ***
Skip If Selenium Disabled
    Run Keyword If    '${SELENIUM_ENABLED}' != 'true'    Skip    Selenium disabled. Set SELENIUM_ENABLED=true and install a WebDriver locally.
