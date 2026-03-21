/**
 * E2E tests for the RSI verification wizard flow.
 *
 * Tests cover: unauthenticated gate, handle validation, the full happy-path
 * wizard (handle → token → confirm → success), and the failure branch.
 *
 * All backend calls are intercepted with page.route() — no real stack needed.
 */
import { test, expect } from '@playwright/test';
import {
  mockStatus,
  mockVerifyStart,
  mockVerifyConfirm,
  STATUS_UNAUTHENTICATED,
  STATUS_AUTHENTICATED,
  STATUS_VERIFIED,
} from '../helpers/mock-api.js';

// Shared token fixture used across several tests.
const MOCK_TOKEN = 'scid:abcdef1234567890abcdef1234567890';
const MOCK_EXPIRES_AT = new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString();

test.describe('Verification wizard — unauthenticated', () => {
  test.beforeEach(async ({ page }) => {
    await mockStatus(page, STATUS_UNAUTHENTICATED);
    await page.goto('/verify');
  });

  test('shows login gate with create and sign-in buttons', async ({ page }) => {
    await expect(page.getByRole('button', { name: /create a scid account/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /sign in with scid/i })).toBeVisible();
  });

  test('does not show the handle input', async ({ page }) => {
    await expect(page.locator('#rsi-handle')).not.toBeVisible();
  });
});

test.describe('Verification wizard — authenticated, not verified', () => {
  test.beforeEach(async ({ page }) => {
    await mockStatus(page, STATUS_AUTHENTICATED);
    await page.goto('/verify');
  });

  test('shows RSI handle input on step 1', async ({ page }) => {
    await expect(page.locator('#rsi-handle')).toBeVisible();
    await expect(page.getByRole('button', { name: /begin verification/i })).toBeVisible();
  });

  test('client-side validation: handle too short', async ({ page }) => {
    await page.locator('#rsi-handle').fill('ab');
    await page.getByRole('button', { name: /begin verification/i }).click();
    await expect(page.getByText(/at least 3/i)).toBeVisible();
  });

  test('client-side validation: handle too long', async ({ page }) => {
    await page.locator('#rsi-handle').fill('a'.repeat(61));
    await page.getByRole('button', { name: /begin verification/i }).click();
    await expect(page.getByText(/at most 60/i)).toBeVisible();
  });

  test('client-side validation: special characters rejected', async ({ page }) => {
    await page.locator('#rsi-handle').fill('bad handle!');
    await page.getByRole('button', { name: /begin verification/i }).click();
    await expect(page.getByText(/only letters/i)).toBeVisible();
  });
});

test.describe('Verification wizard — full happy-path flow', () => {
  test('handle → token → confirm → success', async ({ page }) => {
    await mockStatus(page, STATUS_AUTHENTICATED);
    await mockVerifyStart(page, { token: MOCK_TOKEN, expires_at: MOCK_EXPIRES_AT });
    await mockVerifyConfirm(page, { verified: true, handle: 'TestPilot' });

    await page.goto('/verify');

    // Step 1: enter handle and submit.
    await page.locator('#rsi-handle').fill('TestPilot');
    await page.getByRole('button', { name: /begin verification/i }).click();

    // Step 2: token is displayed.
    const tokenEl = page.locator('code').filter({ hasText: 'scid:' });
    await expect(tokenEl).toBeVisible({ timeout: 5000 });
    await expect(tokenEl).toContainText('scid:abcdef');

    // Copy button is available.
    await expect(page.getByRole('button', { name: /copy token/i })).toBeVisible();

    // Step 3: click "I've added it" → triggers confirm API call.
    await page.getByRole('button', { name: /i've added it/i }).click();

    // Step 4: success heading and handle name are shown.
    await expect(page.getByRole('heading', { name: /verified!/i })).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('TestPilot')).toBeVisible();

    // Post-success links are present.
    await expect(page.getByRole('link', { name: /go to homepage/i })).toBeVisible();
  });
});

test.describe('Verification wizard — failure branch', () => {
  test('shows failure step when token is not in bio', async ({ page }) => {
    await mockStatus(page, STATUS_AUTHENTICATED);
    await mockVerifyStart(page, { token: MOCK_TOKEN, expires_at: MOCK_EXPIRES_AT });
    await mockVerifyConfirm(page, {
      verified: false,
      message: 'Token not found in your RSI profile bio. Make sure it is saved and try again.',
    });

    await page.goto('/verify');
    await page.locator('#rsi-handle').fill('TestPilot');
    await page.getByRole('button', { name: /begin verification/i }).click();

    await expect(page.locator('code').filter({ hasText: 'scid:' })).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: /i've added it/i }).click();

    await expect(page.getByRole('heading', { name: /verification failed/i })).toBeVisible({ timeout: 5000 });
    await expect(page.getByText(/token not found/i)).toBeVisible();

    // Recovery buttons are offered.
    await expect(page.getByRole('button', { name: /try again/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /start over/i })).toBeVisible();
  });

  test('"Try Again" returns to the token step', async ({ page }) => {
    await mockStatus(page, STATUS_AUTHENTICATED);
    await mockVerifyStart(page, { token: MOCK_TOKEN, expires_at: MOCK_EXPIRES_AT });
    await mockVerifyConfirm(page, { verified: false, message: 'Token not found.' });

    await page.goto('/verify');
    await page.locator('#rsi-handle').fill('TestPilot');
    await page.getByRole('button', { name: /begin verification/i }).click();

    await expect(page.locator('code').filter({ hasText: 'scid:' })).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: /i've added it/i }).click();

    await expect(page.getByRole('heading', { name: /verification failed/i })).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: /try again/i }).click();

    // Returns to the token display step (code element visible again).
    await expect(page.locator('code').filter({ hasText: 'scid:' })).toBeVisible();
  });

  test('"Start Over" returns to the handle input step', async ({ page }) => {
    await mockStatus(page, STATUS_AUTHENTICATED);
    await mockVerifyStart(page, { token: MOCK_TOKEN, expires_at: MOCK_EXPIRES_AT });
    await mockVerifyConfirm(page, { verified: false, message: 'Token not found.' });

    await page.goto('/verify');
    await page.locator('#rsi-handle').fill('TestPilot');
    await page.getByRole('button', { name: /begin verification/i }).click();

    await expect(page.locator('code').filter({ hasText: 'scid:' })).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: /i've added it/i }).click();

    await expect(page.getByRole('heading', { name: /verification failed/i })).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: /start over/i }).click();

    // Returns to step 1: handle input visible again.
    await expect(page.locator('#rsi-handle')).toBeVisible();
    await expect(page.getByRole('button', { name: /begin verification/i })).toBeVisible();
  });
});

test.describe('Already-verified user dashboard', () => {
  test('home page shows verified RSI handle and metadata', async ({ page }) => {
    await mockStatus(page, STATUS_VERIFIED);
    await page.goto('/');

    await expect(page.getByRole('heading', { name: 'TestPilot' })).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Verified')).toBeVisible();
    await expect(page.getByText('2020-03-10')).toBeVisible(); // enlisted
    await expect(page.getByText('#1234567')).toBeVisible();  // citizen_record
  });

  test('already-verified user on /verify sees profile, not wizard', async ({ page }) => {
    // When user is already verified the page should not show the handle wizard.
    await mockStatus(page, STATUS_VERIFIED);
    await page.goto('/verify');

    // The handle input should NOT be visible for an already-verified user.
    await expect(page.locator('#rsi-handle')).not.toBeVisible({ timeout: 3000 });
  });
});
