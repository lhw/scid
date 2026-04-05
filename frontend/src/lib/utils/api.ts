export interface OrgEntry {
  sid: string;
  name?: string;
  rank_name?: string;
  is_main?: boolean;
  has_logo?: boolean;
}

export interface VerifyStatus {
  authenticated: boolean;
  verified: boolean;
  admin?: boolean;
  user_id?: string;
  username?: string;
  handle?: string;
  verified_at?: string;
  enlisted?: string;
  citizen_record?: string;
  orgs?: OrgEntry[];
  pending_handle?: string;
  pending_expires_at?: string;
  next_sync_at?: string;
}

export interface StartVerifyResponse {
  token: string;
  expires_at: string;
}

export interface ConfirmVerifyResponse {
  verified: boolean;
  handle?: string;
  message?: string;
}

// throwIfError reads the error body and throws if the response is not OK.
// Only consumes the response body on error, leaving it intact for successful responses.
async function throwIfError(res: Response): Promise<void> {
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: "Unknown error" })) as { error?: string };
    throw new Error(body.error ?? `Status ${res.status}`);
  }
}

// Request a one-use Pocket ID signup token so the frontend can redirect a new
// user to Pocket ID's /signup page. Pass the Cloudflare Turnstile challenge
// response token when captcha validation is enabled on the backend.
export async function getSignupToken(turnstileToken?: string): Promise<string> {
  const res = await fetch("/api/auth/signup-token", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ turnstile_token: turnstileToken ?? "" }),
  });
  await throwIfError(res);
  const body = await res.json() as { token: string };
  return body.token;
}
export async function getVerifyStatus(
  fetch: typeof globalThis.fetch,
): Promise<VerifyStatus> {
  const res = await fetch("/api/verify/status");
  if (!res.ok) throw new Error(`Status ${res.status}`);
  return res.json();
}

// Start a new verification for the given RSI handle.
export async function startVerify(
  handle: string,
): Promise<StartVerifyResponse> {
  const res = await fetch("/api/verify/start", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ handle }),
  });
  await throwIfError(res);
  return res.json();
}

// Confirm verification (check that the token is in the RSI bio).
export async function confirmVerify(): Promise<ConfirmVerifyResponse> {
  const res = await fetch("/api/verify/confirm", {
    method: "POST",
  });
  await throwIfError(res);
  return res.json();
}

// Re-scrape the user's RSI profile and update stored claims (handle, avatar,
// enlisted date, citizen record). Returns the updated status.
export async function refreshVerify(): Promise<VerifyStatus> {
  const res = await fetch("/api/verify/refresh", {
    method: "POST",
  });
  await throwIfError(res);
  return res.json();
}

// Delete the user account from both SCID and Pocket ID. This is irreversible.
export async function deleteAccount(): Promise<void> {
  const res = await fetch("/api/account/delete", {
    method: "POST",
  });
  await throwIfError(res);
}

// ---- OIDC Application Registration ----

export interface AppRegistration {
  id: string;
  client_secret?: string; // only present on create or rotate
  name: string;
  description?: string;
  owner_username?: string; // only present in admin context
  launch_url?: string;
  redirect_uris: string[];
  logout_uris: string[];
  is_public: boolean;
  pkce_required: boolean;
  verified_only: boolean;
  listed: boolean;
  has_logo: boolean;
  status: string; // "pending" | "approved" | "rejected"
  rejection_reason?: string;
  created_at: string;
}

export interface CreateAppRequest {
  name: string;
  description?: string;
  launch_url?: string;
  redirect_uris: string[];
  logout_uris: string[];
  is_public: boolean;
  pkce_required: boolean;
  verified_only: boolean;
  listed?: boolean;
}

export interface DirectoryApp {
  id: string;
  name: string;
  description?: string;
  launch_url: string;
  has_logo: boolean;
  verified_only: boolean;
}

export async function listApps(): Promise<AppRegistration[]> {
  const res = await fetch("/api/apps");
  await throwIfError(res);
  return res.json();
}

export async function listPublicApps(): Promise<DirectoryApp[]> {
  const res = await fetch("/api/apps/directory");
  await throwIfError(res);
  return res.json();
}

export async function createApp(req: CreateAppRequest): Promise<AppRegistration> {
  const res = await fetch("/api/apps", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });
  await throwIfError(res);
  return res.json();
}

export async function getApp(id: string): Promise<AppRegistration> {
  const res = await fetch(`/api/apps/${encodeURIComponent(id)}`);
  await throwIfError(res);
  return res.json();
}

export async function updateApp(id: string, req: Partial<CreateAppRequest>): Promise<AppRegistration> {
  const res = await fetch(`/api/apps/${encodeURIComponent(id)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });
  await throwIfError(res);
  return res.json();
}

export async function deleteApp(id: string): Promise<void> {
  const res = await fetch(`/api/apps/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
  await throwIfError(res);
}

export async function rotateSecret(id: string): Promise<{ client_secret: string }> {
  const res = await fetch(`/api/apps/${encodeURIComponent(id)}/secret`, {
    method: "POST",
  });
  await throwIfError(res);
  return res.json();
}

export async function uploadAppLogo(id: string, file: File): Promise<void> {
  const form = new FormData();
  form.append("file", file);
  const res = await fetch(`/api/apps/${encodeURIComponent(id)}/logo`, {
    method: "PUT",
    body: form,
  });
  await throwIfError(res);
}

// ---- Admin API ----

export async function adminListApps(status?: string): Promise<AppRegistration[]> {
  const url = status ? `/api/admin/apps?status=${encodeURIComponent(status)}` : "/api/admin/apps";
  const res = await fetch(url);
  await throwIfError(res);
  return res.json();
}

export async function adminApproveApp(id: string): Promise<AppRegistration> {
  const res = await fetch(`/api/admin/apps/${encodeURIComponent(id)}/approve`, {
    method: "POST",
  });
  await throwIfError(res);
  return res.json();
}

export async function adminRejectApp(id: string, reason?: string): Promise<AppRegistration> {
  const res = await fetch(`/api/admin/apps/${encodeURIComponent(id)}/reject`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ reason: reason ?? "" }),
  });
  await throwIfError(res);
  return res.json();
}

// ---- Admin: User management ----

export interface AdminUserEntry {
  user_id: string;
  handle: string;
  verified_at: string;
  handle_blocked: boolean;
}

export interface AdminBlockedHandle {
  handle: string;
  blocked_at: string;
  blocked_by: string;
  reason: string;
}

export async function adminListUsers(): Promise<AdminUserEntry[]> {
  const res = await fetch("/api/admin/users");
  await throwIfError(res);
  return res.json();
}

export async function adminDeleteUser(id: string, blockHandle: boolean, reason?: string): Promise<void> {
  const res = await fetch(`/api/admin/users/${encodeURIComponent(id)}`, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ block_handle: blockHandle, reason: reason ?? "" }),
  });
  await throwIfError(res);
}

export async function adminListBlockedHandles(): Promise<AdminBlockedHandle[]> {
  const res = await fetch("/api/admin/handles/blocked");
  await throwIfError(res);
  return res.json();
}

export async function adminBlockHandle(handle: string, reason?: string): Promise<void> {
  const res = await fetch("/api/admin/handles/block", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ handle, reason: reason ?? "" }),
  });
  await throwIfError(res);
}

export async function adminUnblockHandle(handle: string): Promise<void> {
  const res = await fetch(`/api/admin/handles/${encodeURIComponent(handle)}`, {
    method: "DELETE",
  });
  await throwIfError(res);
}

// ---- Admin: Org logo management ----

export interface AdminOrgEntry {
  sid: string;
  name: string;
  has_logo: boolean;
  logo_blocked: boolean;
  fetched_at: string;
}

export async function adminListOrgs(): Promise<AdminOrgEntry[]> {
  const res = await fetch("/api/admin/orgs");
  await throwIfError(res);
  return res.json();
}

export async function adminBlockOrgLogo(sid: string): Promise<void> {
  const res = await fetch(`/api/admin/orgs/${encodeURIComponent(sid)}/block-logo`, {
    method: "POST",
  });
  await throwIfError(res);
}

export async function adminUnblockOrgLogo(sid: string): Promise<void> {
  const res = await fetch(`/api/admin/orgs/${encodeURIComponent(sid)}/block-logo`, {
    method: "DELETE",
  });
  await throwIfError(res);
}

// ---- Admin: Report review queue ----

export interface AdminReport {
  id: string;
  type: "user" | "org";
  target: string;
  reason: string;
  reporter_ip: string;
  created_at: string;
  status: "pending" | "reviewed" | "dismissed";
  reviewed_by?: string;
  reviewed_at?: string;
}

export async function adminListReports(status?: string): Promise<AdminReport[]> {
  const url = status ? `/api/admin/reports?status=${encodeURIComponent(status)}` : "/api/admin/reports";
  const res = await fetch(url);
  await throwIfError(res);
  return res.json();
}

export async function adminReviewReport(id: string): Promise<void> {
  const res = await fetch(`/api/admin/reports/${encodeURIComponent(id)}/review`, { method: "POST" });
  await throwIfError(res);
}

export async function adminDismissReport(id: string): Promise<void> {
  const res = await fetch(`/api/admin/reports/${encodeURIComponent(id)}/dismiss`, { method: "POST" });
  await throwIfError(res);
}

// ---- Public: Report form ----

export async function submitReport(
  type: "user" | "org",
  target: string,
  reason: string,
  turnstileToken?: string,
): Promise<{ id: string }> {
  const res = await fetch("/api/report", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ type, target, reason, turnstile_token: turnstileToken ?? "" }),
  });
  await throwIfError(res);
  return res.json();
}

