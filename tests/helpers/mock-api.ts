/**
 * Shared API mock helpers for Playwright E2E tests.
 *
 * All helpers use page.route() to intercept companion API responses so tests
 * are deterministic and do not require the full Docker Compose stack to run.
 */
import type { Page } from '@playwright/test';

// ------------------------------------------------------------------
// Shared type definitions mirroring the companion API response shapes
// ------------------------------------------------------------------

export interface StatusResponse {
  authenticated: boolean;
  verified: boolean;
  admin?: boolean;
  username?: string;
  user_id?: string;
  handle?: string;
  verified_at?: string;
  enlisted?: string;
  citizen_record?: string;
  pending_handle?: string;
  pending_expires_at?: string;
  orgs?: { sid: string; name: string; is_main: boolean; rank_name: string; has_logo: boolean }[];
  next_sync_at?: string;
}

export interface AppRegistration {
  id: string;
  name: string;
  status: 'pending' | 'approved' | 'rejected';
  owner_user_id: string;
  oidc_client_id: string;
  verified_only: boolean;
  listed: boolean;
  created_at: string;
  rejection_reason: string;
  client_secret?: string;
  launch_url?: string;
  redirect_uris?: string[];
}

// ------------------------------------------------------------------
// Preset status responses
// ------------------------------------------------------------------

export const STATUS_UNAUTHENTICATED: StatusResponse = {
  authenticated: false,
  verified: false,
};

export const STATUS_AUTHENTICATED: StatusResponse = {
  authenticated: true,
  verified: false,
  username: 'testpilot',
  user_id: 'user-test-001',
};

export const STATUS_VERIFIED: StatusResponse = {
  authenticated: true,
  verified: true,
  username: 'testpilot',
  user_id: 'user-test-001',
  handle: 'TestPilot',
  verified_at: '2024-01-15T10:00:00Z',
  enlisted: '2020-03-10',
  citizen_record: '1234567',
};

export const STATUS_ADMIN: StatusResponse = {
  ...STATUS_VERIFIED,
  admin: true,
};

// ------------------------------------------------------------------
// Route registration helpers
// ------------------------------------------------------------------

/** Mock GET /api/verify/status to return a fixed payload. */
export async function mockStatus(page: Page, data: StatusResponse): Promise<void> {
  await page.route('**/api/verify/status', (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(data) }),
  );
}

/** Mock POST /api/verify/start. */
export async function mockVerifyStart(
  page: Page,
  response: { token: string; expires_at: string },
): Promise<void> {
  await page.route('**/api/verify/start', (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(response) }),
  );
}

/** Mock POST /api/verify/confirm. */
export async function mockVerifyConfirm(
  page: Page,
  response: { verified: boolean; handle?: string; message?: string },
): Promise<void> {
  await page.route('**/api/verify/confirm', (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(response) }),
  );
}

/** Mock GET /api/apps and (optionally) POST /api/apps. */
export async function mockApps(
  page: Page,
  list: AppRegistration[],
  onCreate?: AppRegistration,
): Promise<void> {
  await page.route('**/api/apps', (route) => {
    if (route.request().method() === 'POST' && onCreate) {
      return route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify(onCreate),
      });
    }
    return route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(list),
    });
  });
}

/** Mock GET /api/admin/apps (with optional status filter). */
export async function mockAdminApps(page: Page, apps: AppRegistration[]): Promise<void> {
  await page.route('**/api/admin/apps**', (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(apps) }),
  );
}

/** Mock POST /api/admin/apps/{id}/approve. */
export async function mockApproveApp(page: Page, appId: string, result: AppRegistration): Promise<void> {
  await page.route(`**/api/admin/apps/${appId}/approve`, (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(result) }),
  );
}

/** Mock POST /api/admin/apps/{id}/reject. */
export async function mockRejectApp(page: Page, appId: string, result: AppRegistration): Promise<void> {
  await page.route(`**/api/admin/apps/${appId}/reject`, (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(result) }),
  );
}

/** Mock POST /api/account/delete. */
export async function mockDeleteAccount(page: Page): Promise<void> {
  await page.route('**/api/account/delete', (route) =>
    route.fulfill({ status: 204 }),
  );
}

// ------------------------------------------------------------------
// Factory helpers
// ------------------------------------------------------------------

/** Build an AppRegistration fixture. */
export function makeApp(overrides: Partial<AppRegistration> = {}): AppRegistration {
  return {
    id: 'app-test-001',
    name: 'Test Application',
    status: 'pending',
    owner_user_id: 'user-test-001',
    oidc_client_id: 'client-test-001',
    verified_only: false,
    listed: false,
    created_at: new Date().toISOString(),
    rejection_reason: '',
    redirect_uris: ['https://example.com/callback'],
    ...overrides,
  };
}
