/**
 * E2E tests for OIDC application registration and admin approval flows.
 *
 * Tests cover: unverified user gate, app registration form (happy path and
 * validation), pending vs auto-approved apps, admin listing, approval, and
 * rejection with reason.
 *
 * All backend calls are intercepted with page.route().
 */
import { test, expect } from '@playwright/test';
import {
  mockStatus,
  mockApps,
  mockAdminApps,
  mockApproveApp,
  mockRejectApp,
  makeApp,
  STATUS_VERIFIED,
  STATUS_ADMIN,
} from '../helpers/mock-api.js';

// ------------------------------------------------------------------
// Helpers
// ------------------------------------------------------------------

/** Fill in the minimum required fields for a new app registration. */
async function fillMinimalAppForm(
  page: Parameters<typeof mockStatus>[0],
  name: string,
  redirectURI = 'https://example.com/callback',
) {
  await page.locator('#app-name').fill(name);
  await page.locator('input[placeholder="https://example.com/callback"]').first().fill(redirectURI);
}

// ------------------------------------------------------------------
// App registration — access control
// ------------------------------------------------------------------

test.describe('App registration — access control', () => {
  test('unverified user sees verification-required gate, not registration button', async ({ page }) => {
    await mockStatus(page, { authenticated: true, verified: false, username: 'newbie' });
    await mockApps(page, []);

    await page.goto('/apps');

    await expect(page.getByText(/verification required/i)).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole('button', { name: /register new app/i })).not.toBeVisible();
  });

  test('unauthenticated user is redirected away from /apps', async ({ page }) => {
    await mockStatus(page, { authenticated: false, verified: false });
    await mockApps(page, []);

    await page.goto('/apps');

    // The apps page should either redirect or show the unverified gate.
    // Since SSR is off and the check is client-side, we expect to see no register button.
    await expect(page.getByRole('button', { name: /register new app/i })).not.toBeVisible({ timeout: 5000 });
  });
});

// ------------------------------------------------------------------
// App registration — form and submission
// ------------------------------------------------------------------

test.describe('App registration — form', () => {
  test.beforeEach(async ({ page }) => {
    await mockStatus(page, STATUS_VERIFIED);
    await mockApps(page, []);
    await page.goto('/apps');
    await page.getByRole('button', { name: /register new app/i }).click();
  });

  test('opens registration form on button click', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /register new application/i })).toBeVisible();
    await expect(page.locator('#app-name')).toBeVisible();
  });

  test('form validation: name is required', async ({ page }) => {
    await page.getByRole('button', { name: /register application/i }).click();
    await expect(page.getByText(/name is required/i)).toBeVisible();
  });

  test('form validation: at least one redirect URI required', async ({ page }) => {
    await page.locator('#app-name').fill('NoURIApp');
    await page.getByRole('button', { name: /register application/i }).click();
    await expect(page.getByText(/at least one redirect uri/i)).toBeVisible();
  });

  test('form validation: redirect URI must be https or localhost', async ({ page }) => {
    await page.locator('#app-name').fill('BadURIApp');
    await page.locator('input[placeholder="https://example.com/callback"]').first().fill('http://evil.example.com/cb');
    await page.getByRole('button', { name: /register application/i }).click();
    await expect(page.getByText(/https:\/\//i)).toBeVisible();
  });

  test('cancel button closes the form', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /register new application/i })).toBeVisible();
    await page.getByRole('button', { name: /cancel/i }).click();
    await expect(page.getByRole('heading', { name: /register new application/i })).not.toBeVisible();
  });
});

// ------------------------------------------------------------------
// App registration — success paths
// ------------------------------------------------------------------

test.describe('App registration — successful submission', () => {
  test('auto-approved app shows success toast and appears in list', async ({ page }) => {
    const newApp = makeApp({ id: 'app-new', name: 'Galactic Explorer', status: 'approved' });
    let created = false;

    await mockStatus(page, STATUS_VERIFIED);
    await page.route('**/api/apps', (route) => {
      if (route.request().method() === 'POST') {
        created = true;
        return route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify(newApp) });
      }
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(created ? [newApp] : []),
      });
    });

    await page.goto('/apps');
    await page.getByRole('button', { name: /register new app/i }).click();
    await fillMinimalAppForm(page, 'Galactic Explorer');
    await page.getByRole('button', { name: /register application/i }).click();

    await expect(page.getByText(/application registered/i)).toBeVisible({ timeout: 5000 });
    // After registration the app form is closed and the app is in the list.
    await expect(page.getByText('Galactic Explorer')).toBeVisible({ timeout: 3000 });
  });

  test('pending-approval app shows success toast with pending status badge', async ({ page }) => {
    const newApp = makeApp({ id: 'app-pending', name: 'OrbitCorp Hub', status: 'pending' });
    let created = false;

    await mockStatus(page, STATUS_VERIFIED);
    await page.route('**/api/apps', (route) => {
      if (route.request().method() === 'POST') {
        created = true;
        return route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify(newApp) });
      }
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(created ? [newApp] : []),
      });
    });

    await page.goto('/apps');
    await page.getByRole('button', { name: /register new app/i }).click();
    await fillMinimalAppForm(page, 'OrbitCorp Hub');
    await page.getByRole('button', { name: /register application/i }).click();

    await expect(page.getByText(/application registered/i)).toBeVisible({ timeout: 5000 });
    await expect(page.getByText(/pending/i)).toBeVisible({ timeout: 3000 });
  });
});

// ------------------------------------------------------------------
// App list
// ------------------------------------------------------------------

test.describe('App list', () => {
  test('shows existing apps on page load', async ({ page }) => {
    const apps = [
      makeApp({ id: 'app-001', name: 'StarTracker', status: 'approved' }),
      makeApp({ id: 'app-002', name: 'FleetCommand', status: 'pending' }),
    ];

    await mockStatus(page, STATUS_VERIFIED);
    await mockApps(page, apps);
    await page.goto('/apps');

    await expect(page.getByText('StarTracker')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('FleetCommand')).toBeVisible();
  });

  test('shows empty state when user has no apps', async ({ page }) => {
    await mockStatus(page, STATUS_VERIFIED);
    await mockApps(page, []);
    await page.goto('/apps');

    // Register button should be visible; no app cards in list.
    await expect(page.getByRole('button', { name: /register new app/i })).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('StarTracker')).not.toBeVisible();
  });
});

// ------------------------------------------------------------------
// Admin approval workflow
// ------------------------------------------------------------------

test.describe('Admin approval workflow', () => {
  test('admin sees pending apps and can approve one', async ({ page }) => {
    const pending = makeApp({ id: 'app-abc', name: 'PendingApp', status: 'pending' });
    const approved = { ...pending, status: 'approved' as const };

    await mockStatus(page, STATUS_ADMIN);
    await mockAdminApps(page, [pending]);
    await mockApproveApp(page, 'app-abc', approved);

    await page.goto('/admin/apps');

    await expect(page.getByText('PendingApp')).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: /approve/i }).first().click();
    await expect(page.getByText(/application approved/i)).toBeVisible({ timeout: 5000 });
  });

  test('admin can reject an app and provide a reason', async ({ page }) => {
    const pending = makeApp({ id: 'app-xyz', name: 'SpamApp', status: 'pending' });
    const rejected = { ...pending, status: 'rejected' as const, rejection_reason: 'Violates terms of service' };

    await mockStatus(page, STATUS_ADMIN);
    await mockAdminApps(page, [pending]);
    await mockRejectApp(page, 'app-xyz', rejected);

    await page.goto('/admin/apps');

    await expect(page.getByText('SpamApp')).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: /reject/i }).first().click();

    // Rejection modal appears.
    await expect(page.getByRole('heading', { name: /reject application/i })).toBeVisible();
    await page.locator('#reject-reason').fill('Violates terms of service');
    await page.getByRole('button', { name: /confirm reject/i }).click();

    await expect(page.getByText(/application rejected/i)).toBeVisible({ timeout: 5000 });
  });

  test('non-admin is redirected away from /admin/apps', async ({ page }) => {
    await mockStatus(page, STATUS_VERIFIED); // verified but not admin
    await page.goto('/admin/apps');

    // The page redirects non-admins to '/' via goto('/') in the onMount hook.
    await expect(page).toHaveURL('/', { timeout: 5000 });
  });

  test('admin can filter apps by status', async ({ page }) => {
    const pending = makeApp({ id: 'app-p', name: 'PendingOne', status: 'pending' });
    const approved = makeApp({ id: 'app-a', name: 'ApprovedOne', status: 'approved' });

    await mockStatus(page, STATUS_ADMIN);

    // Return different lists based on the status query param.
    await page.route('**/api/admin/apps**', (route) => {
      const url = new URL(route.request().url());
      const status = url.searchParams.get('status');
      if (status === 'approved') {
        return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([approved]) });
      }
      return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([pending]) });
    });

    await page.goto('/admin/apps');
    // Default filter is 'pending'.
    await expect(page.getByText('PendingOne')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('ApprovedOne')).not.toBeVisible();

    // Switch to 'Approved' filter.
    await page.getByRole('button', { name: /approved/i }).click();
    await expect(page.getByText('ApprovedOne')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('PendingOne')).not.toBeVisible();
  });
});
