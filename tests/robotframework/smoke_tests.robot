*** Settings ***
Documentation    Fast smoke checks — no external services required.
Resource         resources/common.robot
Suite Setup      Log Suite Context

*** Test Cases ***
Python Runtime Is Available
    [Documentation]    Verifies the test runner environment is functional.
    ${version}=    Evaluate    sys.version    modules=sys
    Should Match Regexp    ${version}    \\d+\\.\\d+

Environment Variables Are Readable
    [Documentation]    CI injects secrets; locally defaults are fine.
    ${home}=    Get Environment Variable    HOME    default=${EMPTY}
    ${user}=    Get Environment Variable    USERNAME    default=ci-user
    Should Not Be Empty    ${user}

String And List Operations
    [Documentation]    Basic Robot keyword sanity.
    @{items}=    Create List    smoke    api    ui
    Length Should Be    ${items}    3
    Should Contain    ${items}    smoke

Numeric Assertions
    ${sum}=    Evaluate    2 + 2
    Should Be Equal As Integers    ${sum}    4
