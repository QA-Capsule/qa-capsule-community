*** Settings ***
Documentation    HTTP health checks via RequestsLibrary (public httpbin by default).
Resource         resources/common.robot
Library          RequestsLibrary
Suite Setup      Log Suite Context
Test Setup       Create Session    api    https://${API_HOST}    verify=${VERIFY_SSL}

*** Variables ***
${API_HOST}       %{API_HEALTH_HOST=jsonplaceholder.typicode.com}
${VERIFY_SSL}     ${True}

*** Test Cases ***
GET Resource Returns 200
    [Documentation]    Confirms a public REST endpoint responds with HTTP 200.
    [Tags]    api    smoke
    ${response}=    GET On Session    api    /todos/1
    Should Be Valid HTTP Status    ${response.status_code}    200
    Dictionary Should Contain Key    ${response.json()}    id

GET Collection Is Non Empty
    [Documentation]    Validates list endpoints return data.
    [Tags]    api
    ${response}=    GET On Session    api    /users
    Should Be Valid HTTP Status    ${response.status_code}    200
    ${users}=    Set Variable    ${response.json()}
    Should Not Be Empty    ${users}

POST Creates Resource
    [Documentation]    Sends JSON to a mock create endpoint (201 expected).
    [Tags]    api
    ${headers}=    Create Dictionary    Content-Type=application/json
    ${body}=    Create Dictionary    title=QA Capsule    body=robot api test    userId=1
    ${response}=    POST On Session    api    /posts    json=${body}    headers=${headers}
    Should Be Valid HTTP Status    ${response.status_code}    201
    Should Be Equal As Integers    ${response.json()}[userId]    1
