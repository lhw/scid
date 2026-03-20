import { test, expect } from '@playwright/test';

test.describe('Home page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('page loads and has correct title', async ({ page }) => {
    await expect(page).toHaveTitle(/SCID/i);
  });

  test('header shows SCID logo with teal colour', async ({ page }) => {
    const logo = page.getByRole('link', { name: /SCID/i }).first();
    await expect(logo).toBeVisible();
    // The SCID text should be rendered in teal (#00d4ff)
    const color = await logo.locator('span').first().evaluate((el) => {
      return window.getComputedStyle(el).color;
    });
    // rgb(0, 212, 255) == #00d4ff
    expect(color).toBe('rgb(0, 212, 255)');
  });

  test('header has Verify Identity nav link', async ({ page }) => {
    const verifyLink = page.getByRole('link', { name: /verify identity/i });
    await expect(verifyLink).toBeVisible();
    await expect(verifyLink).toHaveAttribute('href', '/verify');
  });

  test('page background is dark navy', async ({ page }) => {
    const bg = await page.locator('body').evaluate((el) => {
      return window.getComputedStyle(el).backgroundColor;
    });
    // rgb(10, 14, 26) == #0a0e1a
    expect(bg).toBe('rgb(10, 14, 26)');
  });

  test('hero section is visible', async ({ page }) => {
    // The hero contains a heading with "Star Citizen" or "SCID"
    const heading = page.getByRole('heading', { level: 1 });
    await expect(heading).toBeVisible();
  });

  test('CTA button links to /verify', async ({ page }) => {
    const cta = page.getByRole('link', { name: /verify/i }).filter({ hasText: /verify/i }).first();
    await expect(cta).toBeVisible();
  });

  test('footer shows unofficial fansite disclaimer', async ({ page }) => {
    const footer = page.locator('footer');
    await expect(footer).toContainText(/unofficial/i);
  });

  test('CSS is applied — body text is not black (default browser style)', async ({ page }) => {
    const color = await page.locator('body').evaluate((el) => {
      return window.getComputedStyle(el).color;
    });
    // Should be slate-100 / #e2e8f0, not black (rgb(0,0,0))
    expect(color).not.toBe('rgb(0, 0, 0)');
  });
});
