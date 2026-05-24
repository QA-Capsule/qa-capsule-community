*** Settings ***
Documentation    Intentional failure for QA Capsule alert demos (CI / manual workflow).
Resource         resources/common.robot
Suite Setup      Log Suite Context

*** Test Cases ***
Simulated Payment Gateway Rejection
    [Documentation]    Fails on purpose so JUnit contains a failure and QA Capsule can raise an incident.
    [Tags]    demo    intentional-failure
    Log    [STDOUT] Sending malformed payload to payment gateway...    console=yes
    ${status}=    Set Variable    ${400}
    Should Be Equal As Integers    ${status}    ${201}    Expected HTTP 201 Created, got ${status}
