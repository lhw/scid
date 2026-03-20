import { test, expect } from '@playwright/test';

test.describe('Verification wizard page — unauthenticated', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/verify');
  });

  test('page loads without crashing', async ({ page }) => {
    await expect(page.locator('body')).not.toBeEmpty();
    const bg = await page.locator('body').evaluate((el) => {
      return window.getComputedStyle(el).backgroundColor;
    });
    expect(bg).toBe('rgb(10, 14, 26)');
  });

  test('shows login gate when not authenticated', async ({ page }) => {
    // Both "create account" and "sign in" options should be present
    await expect(page.getByRole('button', { name: /create a scid account/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /sign in with scid/i })).toBeVisible();
  });

  test('header is present on verify page', async ({ page }) => {
    const header = page.locator('header');
    await expect(header).toBeVisible();
    await expect(header).toContainText('SCID');
  });
});

test.describe('Verification wizard page — authenticated', () => {
  test.beforeEach(async ({ page }) => {
    // Inject a fake access token so the login gate is bypassed.
    // The API calls will fail (fake token), but the wizard UI should render.
    await page.addInitScript(() => {
      sessionStorage.setItem('scid_access_token', 'fake-token-for-ui-test');
    });
    await page.goto('/verify');
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

    await input.fill('ab');
    await btn.click();

    const error = page.locator('[role="alert"], .text-red-400, .text-red-500').first();
    await expect(error).toBeVisible();
  });
});
