const { test, expect } = require("@playwright/test");

test.describe("Playwright sample suite", () => {
  test("homepage responds", async ({ page }) => {
    await page.goto("https://example.com");
    await expect(page).toHaveTitle(/Example Domain/i);
  });

  test("intentional failure for demo", async ({ page }) => {
    await page.goto("https://example.com");
    await expect(page.locator("#does-not-exist")).toBeVisible();
  });
});
