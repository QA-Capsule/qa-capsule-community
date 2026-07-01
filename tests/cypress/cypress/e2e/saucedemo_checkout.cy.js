/**
 * E2E checkout flow against Swag Labs (saucedemo.com).
 *
 * Target: https://www.saucedemo.com/
 * Credentials: standard_user / secret_sauce
 *
 *   TC-01  Inventory page after login     → passes
 *   TC-02  Add item to cart               → passes
 *   TC-03  Cart shows selected item       → passes
 *   TC-04  Checkout with broken locator   → FAILS (MCP self-healing demo)
 *
 * On failure, cypress/support/qa-capsule.js captures page HTML for heal_incident.
 */

const BASE_URL = 'https://www.saucedemo.com/';
const USERNAME = 'standard_user';
const PASSWORD = 'secret_sauce';

// Real checkout button: [data-test="checkout"]
const BROKEN_CHECKOUT = '[data-test="proceed-to-payment"]';

describe('Swag Labs Checkout — QA Capsule Self-Healing Demo', () => {
  beforeEach(() => {
    cy.visit(BASE_URL);
    cy.get('#user-name').type(USERNAME);
    cy.get('#password').type(PASSWORD);
    cy.get('#login-button').click();
    cy.get('.title').should('contain.text', 'Products');
  });

  it('TC-01 Inventory lists products after login', () => {
    cy.get('.inventory_item').should('have.length.at.least', 1);
    cy.get('[data-test="add-to-cart-sauce-labs-backpack"]').should('be.visible');
  });

  it('TC-02 Add backpack to cart', () => {
    cy.get('[data-test="add-to-cart-sauce-labs-backpack"]').click();
    cy.get('.shopping_cart_badge').should('contain.text', '1');
  });

  it('TC-03 Cart page lists the backpack', () => {
    cy.get('[data-test="add-to-cart-sauce-labs-backpack"]').click();
    cy.get('.shopping_cart_link').click();
    cy.get('.inventory_item_name').should('contain.text', 'Sauce Labs Backpack');
  });

  it('TC-04 Checkout with broken locator [self-healing demo]', () => {
    cy.get('[data-test="add-to-cart-sauce-labs-backpack"]').click();
    cy.get('.shopping_cart_link').click();
    // Intentionally wrong — real selector is [data-test="checkout"]
    cy.get(BROKEN_CHECKOUT).click();
    cy.get('[data-test="firstName"]').should('be.visible');
  });
});
