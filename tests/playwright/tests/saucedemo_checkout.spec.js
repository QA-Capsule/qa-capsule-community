/**
 * E2E checkout flow against Swag Labs (saucedemo.com).
 *
 * Target: https://www.saucedemo.com/
 * Credentials: standard_user / secret_sauce
 *
 *   TC-01..TC-03  Login + cart flow  → pass
 *   TC-04         Broken checkout    → FAIL (MCP self-healing demo)
 */

const { test, expect } = require('../fixtures/qa-capsule-fixture');

const BASE_URL = 'https://www.saucedemo.com/';
const USERNAME = 'standard_user';
const PASSWORD = 'secret_sauce';
const BROKEN_CHECKOUT = '[data-test="proceed-to-payment"]';

async function login(page) {
  await page.goto(BASE_URL);
  await page.fill('#user-name', USERNAME);
  await page.fill('#password', PASSWORD);
  await page.click('#login-button');
  await expect(page.locator('.title')).toContainText('Products');
}

test.describe('Swag Labs Checkout — QA Capsule Self-Healing Demo', () => {
  test('TC-01 Inventory lists products after login', async ({ page }) => {
    await login(page);
    await expect(page.locator('.inventory_item').first()).toBeVisible();
  });

  test('TC-02 Add backpack to cart', async ({ page }) => {
    await login(page);
    await page.click('[data-test="add-to-cart-sauce-labs-backpack"]');
    await expect(page.locator('.shopping_cart_badge')).toHaveText('1');
  });

  test('TC-03 Cart page lists the backpack', async ({ page }) => {
    await login(page);
    await page.click('[data-test="add-to-cart-sauce-labs-backpack"]');
    await page.click('.shopping_cart_link');
    await expect(page.locator('.inventory_item_name')).toContainText('Sauce Labs Backpack');
  });

  test('TC-04 Checkout with broken locator [self-healing demo]', async ({ page }) => {
    await login(page);
    await page.click('[data-test="add-to-cart-sauce-labs-backpack"]');
    await page.click('.shopping_cart_link');
    await page.locator(BROKEN_CHECKOUT).click({ timeout: 5000 });
    await expect(page.locator('[data-test="firstName"]')).toBeVisible();
  });
});
