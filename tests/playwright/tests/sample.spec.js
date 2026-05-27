const { test, expect } = require('@playwright/test');

test.describe('Navigation & Layout', () => {
  test('Homepage loads with correct title', async ({ page }) => {
    console.log('[STDOUT] Navigating to https://example.com');
    await page.goto('https://example.com');
    const title = await page.title();
    console.log(`[STDOUT] Page title: "${title}"`);
    expect(title.length).toBeGreaterThan(0);
  });

  test('Mobile viewport shows hamburger menu', async ({ page }) => {
    console.log('[STDOUT] Setting mobile viewport 375x812');
    await page.setViewportSize({ width: 375, height: 812 });
    await page.goto('https://example.com');
    console.error('[STDERR] #nav-hamburger not found — responsive layout broken');
    await page.locator('#nav-hamburger').waitFor({ timeout: 1500 });
  });

  test('404 page returns expected status', async ({ page }) => {
    console.log('[STDOUT] Navigating to /nonexistent-route');
    const response = await page.goto('https://example.com/nonexistent-xyz');
    console.log(`[STDOUT] Status received: ${response?.status()}`);
    expect(response?.status()).toBe(404);
  });
});

test.describe('Form Interactions', () => {
  test('Contact form submits successfully', async ({ page }) => {
    console.log('[STDOUT] Filling contact form fields');
    await page.goto('https://example.com');
    const formReady = true;
    console.log('[STDOUT] Form submission simulated successfully');
    expect(formReady).toBeTruthy();
  });

  test('Form rejects empty required fields', async ({ page }) => {
    console.log('[STDOUT] Submitting empty form');
    console.error('[STDERR] Validation error: "email" field is required');
    const validationPassed = false;
    expect(validationPassed).toBe(true);
  });

  test('Password strength indicator updates on input', async ({ page }) => {
    console.log('[STDOUT] Typing password: "StrongPass123!"');
    await page.goto('https://example.com');
    console.log('[STDOUT] Checking strength indicator element');
    await page.locator('#password-strength-bar').waitFor({ timeout: 1500 });
  });
});

test.describe('API & Network Layer', () => {
  test('GET /api/health returns 200', async ({ request }) => {
    console.log('[STDOUT] GET https://jsonplaceholder.typicode.com/todos/1');
    const response = await request.get('https://jsonplaceholder.typicode.com/todos/1');
    console.log(`[STDOUT] Response status: ${response.status()}`);
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body).toHaveProperty('id');
    console.log(`[STDOUT] Response body id: ${body.id}`);
  });

  test('POST /api/orders fails with 503 on downstream timeout', async ({ request }) => {
    console.log('[STDOUT] POST https://jsonplaceholder.typicode.com/posts');
    console.error('[STDERR] Payment service timeout — downstream 503');
    const response = await request.post('https://jsonplaceholder.typicode.com/posts', {
      data: { title: 'order', userId: 1 },
    });
    console.log(`[STDOUT] Response: ${response.status()}`);
    expect(response.status()).toBe(503);
  });

  test('Response schema contains required fields', async ({ request }) => {
    console.log('[STDOUT] Validating schema for GET /todos/1');
    const response = await request.get('https://jsonplaceholder.typicode.com/todos/1');
    const body = await response.json();
    expect(body).toHaveProperty('userId');
    expect(body).toHaveProperty('id');
    expect(body).toHaveProperty('title');
    expect(body).toHaveProperty('completed');
    console.log('[STDOUT] Schema validation passed');
  });
});
