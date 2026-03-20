import { test, expect } from '@playwright/test';

test.describe('Verification wizard page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/verify');
  });

  test('page loads without crashing', async ({ page }) => {
    // Should not show a Svelte error overlay or blank white page
    await expect(page.locator('body')).not.toBeEmpty();
    const bg = await page.locator('body').evaluate((el) => {
      return window.getComputedStyle(el).backgroundColor;
    });
    expect(bg).toBe('rgb(10, 14, 26)');
  });

  test('shows RSI handle input in step 1', async ({ page }) => {
    const input = page.getByPlaceholder(/CyFreeze/i);
    await expect(input).toBeVisible();
  });

  test('shows Begin Verification button', async ({ page }) => {
    const btn = page.getByRole('button', { name: /begin verification/i });
    await expect(btn).toBeVisible();
  });

  test('handle input validates minimum length', async ({ page }) => {
    const input = page.getByPlaceholder(/CyFreeze/i);
    const btn = page.getByRole('button', { name: /begin verification/i });

    // Type a too-short handle and submit
    await input.fill('ab');
    await btn.click();

    // Should show an inline validation error
    const error = page.locator('[role="alert"], .text-red-400, .text-red-500').first();
    await expect(error).toBeVisible();
  });

  test('header is present on verify page too', async ({ page }) => {
    const header = page.locator('header');
    await expect(header).toBeVisible();
    await expect(header).toContainText('SCID');
  });
});
