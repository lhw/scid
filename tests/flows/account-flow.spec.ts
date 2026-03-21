/**
 * E2E tests for account management flows.
 *
 * Tests cover: account deletion (with confirmation dialog), the "refresh RSI
 * info" action, and the "change RSI handle" flow entry point.
 *
 * All backend calls are intercepted with page.route().
 */
import { test, expect } from '@playwright/test';
import { mockStatus, mockDeleteAccount, STATUS_VERIFIED } from '../helpers/mock-api.js';

// ------------------------------------------------------------------
// Account deletion
// ------------------------------------------------------------------

test.describe('Account deletion', () => {
  test.beforeEach(async ({ page }) => {
    await mockStatus(page, STATUS_VERIFIED);
    await mockDeleteAccount(page);
    await page.goto('/');
  });

  test('delete button initially hidden behind a confirmation trigger', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'TestPilot' })).toBeVisible({ timeout: 5000 });

    // "Yes, delete" confirm button should not be visible before the user clicks "Delete account".
    await expect(page.getByRole('button', { name: /yes, delete/i })).not.toBeVisible();
  });

  test('clicking "Delete account" shows the confirmation panel', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'TestPilot' })).toBeVisible({ timeout: 5000 });

    await page.getByRole('button', { name: /delete account/i }).click();

    await expect(page.getByText(/this will permanently delete/i)).toBeVisible();
    await expect(page.getByRole('button', { name: /yes, delete/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /cancel/i })).toBeVisible();
  });

  test('"Cancel" dismisses the confirmation panel', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'TestPilot' })).toBeVisible({ timeout: 5000 });

    await page.getByRole('button', { name: /delete account/i }).click();
    await expect(page.getByRole('button', { name: /yes, delete/i })).toBeVisible();

    await page.getByRole('button', { name: /cancel/i }).click();

    // Confirmation panel should be gone; the trigger link should be visible again.
    await expect(page.getByRole('button', { name: /yes, delete/i })).not.toBeVisible();
    await expect(page.getByRole('button', { name: /delete account/i })).toBeVisible();
  });

  test('"Yes, delete" calls the delete API and shows success toast', async ({ page }) => {
    // Track whether the delete endpoint was called.
    let deleteCalled = false;
    await page.route('**/api/account/delete', (route) => {
      deleteCalled = true;
      return route.fulfill({ status: 204 });
    });

    await expect(page.getByRole('heading', { name: 'TestPilot' })).toBeVisible({ timeout: 5000 });

    await page.getByRole('button', { name: /delete account/i }).click();
    await page.getByRole('button', { name: /yes, delete/i }).click();

    await expect(page.getByText(/account deleted/i)).toBeVisible({ timeout: 5000 });
    expect(deleteCalled).toBe(true);
  });
});

// ------------------------------------------------------------------
// Refresh RSI info
// ------------------------------------------------------------------

test.describe('Refresh RSI info', () => {
  test('clicking refresh calls the refresh API and shows success toast', async ({ page }) => {
    let refreshCalled = false;
    await mockStatus(page, STATUS_VERIFIED);
    await page.route('**/api/verify/refresh', (route) => {
      refreshCalled = true;
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ ...STATUS_VERIFIED, enlisted: '2020-04-01' }),
      });
    });

    await page.goto('/');
    await expect(page.getByRole('heading', { name: 'TestPilot' })).toBeVisible({ timeout: 5000 });

    await page.getByRole('button', { name: /refresh rsi info/i }).click();
    await expect(page.getByText(/rsi profile refreshed/i)).toBeVisible({ timeout: 5000 });
    expect(refreshCalled).toBe(true);
  });
});

// ------------------------------------------------------------------
// Change RSI handle entry
// ------------------------------------------------------------------

test.describe('Change RSI handle', () => {
  test('clicking "Change RSI handle" shows a warning with a Continue link to /verify', async ({ page }) => {
    await mockStatus(page, STATUS_VERIFIED);

    await page.goto('/');
    await expect(page.getByRole('heading', { name: 'TestPilot' })).toBeVisible({ timeout: 5000 });

    await page.getByRole('button', { name: /change rsi handle/i }).click();

    await expect(page.getByText(/replaces your current rsi handle/i)).toBeVisible();
    await expect(page.getByRole('link', { name: /continue/i })).toHaveAttribute('href', '/verify');
  });
});
