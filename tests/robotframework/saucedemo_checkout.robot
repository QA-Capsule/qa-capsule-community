*** Settings ***
Documentation
...    E2E checkout on Swag Labs (saucedemo.com) — 3 passing tests + 1 broken locator for MCP self-healing.
...
...    Credentials: standard_user / secret_sauce
...    Real checkout button: css=[data-test="checkout"]
...    Broken locator (TC-04): css=[data-test="proceed-to-payment"]

Resource    resources/common.robot
Library     Browser

Suite Setup      New Browser    chromium    headless=true
Suite Teardown   Close Browser

*** Variables ***
${BASE_URL}         https://www.saucedemo.com/
${VALID_USER}       standard_user
${VALID_PASS}       secret_sauce
${BROKEN_CHECKOUT}  css=[data-test="proceed-to-payment"]

*** Keywords ***
Login To Inventory
    New Page    ${BASE_URL}
    Fill Text    id=user-name    ${VALID_USER}
    Fill Text    id=password    ${VALID_PASS}
    Click    id=login-button
    Get Text    css=.title    contains    Products

*** Test Cases ***
TC-01 Inventory Lists Products After Login
    [Documentation]    Confirms login lands on the product catalog.
    [Tags]    smoke    saucedemo    passing
    Login To Inventory
    Get Element    css=.inventory_item >> nth=0
    Close Page

TC-02 Add Backpack To Cart
    [Documentation]    Adds one item and checks the cart badge.
    [Tags]    saucedemo    passing
    Login To Inventory
    Click    css=[data-test="add-to-cart-sauce-labs-backpack"]
    Get Text    css=.shopping_cart_badge    ==    1
    Close Page

TC-03 Cart Shows Selected Item
    [Documentation]    Opens the cart and verifies the backpack line item.
    [Tags]    saucedemo    passing
    Login To Inventory
    Click    css=[data-test="add-to-cart-sauce-labs-backpack"]
    Click    css=.shopping_cart_link
    Get Text    css=.inventory_item_name    contains    Sauce Labs Backpack
    Close Page

TC-04 Checkout With Broken Locator (Self-Healing Demo)
    [Documentation]
    ...    Fails on purpose — selector does not exist on the cart page.
    ...    dom_capture_listener stores HTML for heal_incident MCP.
    [Tags]    saucedemo    broken-locator    self-healing-demo
    Login To Inventory
    Click    css=[data-test="add-to-cart-sauce-labs-backpack"]
    Click    css=.shopping_cart_link
    Click    ${BROKEN_CHECKOUT}
    Get Element    css=[data-test="firstName"]
    Close Page
