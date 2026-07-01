*** Settings ***
Documentation    Live REST checks against reqres.in (public demo API).
Resource         resources/common.robot
Library          RequestsLibrary
Suite Setup      Log Suite Context
Test Setup       Create Session    api    https://${API_HOST}    verify=${VERIFY_SSL}

*** Variables ***
${API_HOST}       %{API_HEALTH_HOST=reqres.in}
${VERIFY_SSL}     ${True}

*** Test Cases ***
GET Users Page Returns 200
    [Documentation]    Paginated user list must return data array.
    [Tags]    api    smoke    passing
    ${response}=    GET On Session    api    /api/users?page=1
    Should Be Valid HTTP Status    ${response.status_code}    200
    Dictionary Should Contain Key    ${response.json()}    data
    Should Not Be Empty    ${response.json()}[data]

GET User By Id Returns Profile
    [Documentation]    Single user endpoint returns id and email.
    [Tags]    api    passing
    ${response}=    GET On Session    api    /api/users/2
    Should Be Valid HTTP Status    ${response.status_code}    200
    Should Be Equal As Integers    ${response.json()}[data][id]    2
    Dictionary Should Contain Key    ${response.json()}[data]    email

POST Create User Returns 201
    [Documentation]    Create endpoint accepts JSON body.
    [Tags]    api    passing
    ${headers}=    Create Dictionary    Content-Type=application/json
    ${body}=    Create Dictionary    name=QA Capsule    job=SRE Engineer
    ${response}=    POST On Session    api    /api/users    json=${body}    headers=${headers}
    Should Be Valid HTTP Status    ${response.status_code}    201
    Should Be Equal    ${response.json()}[name]    QA Capsule

GET Missing User Wrong Status Expectation (Self-Healing Demo)
    [Documentation]
    ...    Intentional failure: /api/users/23 returns 404 but test expects 200.
    ...    Exercises API failure ingestion and MCP healing gate.
    [Tags]    api    broken-contract    self-healing-demo
    ${response}=    GET On Session    api    /api/users/23    expected_status=404
    Should Be Valid HTTP Status    ${response.status_code}    200
