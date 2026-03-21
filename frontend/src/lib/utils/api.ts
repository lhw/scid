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

// Request a one-use Pocket ID signup token so the frontend can redirect a new
// user to Pocket ID's /signup page. Pass the Cloudflare Turnstile challenge
// response token when captcha validation is enabled on the backend.
export async function getSignupToken(turnstileToken?: string): Promise<string> {
  const res = await fetch("/api/auth/signup-token", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ turnstile_token: turnstileToken ?? "" }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error ?? `Status ${res.status}`);
  }
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
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error ?? `Status ${res.status}`);
  }
  return res.json();
}

// Confirm verification (check that the token is in the RSI bio).
export async function confirmVerify(): Promise<ConfirmVerifyResponse> {
  const res = await fetch("/api/verify/confirm", {
    method: "POST",
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error ?? `Status ${res.status}`);
  }
  return res.json();
}

// Re-scrape the user's RSI profile and update stored claims (handle, avatar,
// enlisted date, citizen record). Returns the updated status.
export async function refreshVerify(): Promise<VerifyStatus> {
  const res = await fetch("/api/verify/refresh", {
    method: "POST",
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error ?? `Status ${res.status}`);
  }
  return res.json();
}

// Delete the user account from both SCID and Pocket ID. This is irreversible.
export async function deleteAccount(): Promise<void> {
  const res = await fetch("/api/account/delete", {
    method: "POST",
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error ?? `Status ${res.status}`);
  }
}

// ---- OIDC Application Registration ----

export interface AppRegistration {
  id: string;
  client_secret?: string; // only present on create or rotate
  name: string;
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
  launch_url: string;
  has_logo: boolean;
  verified_only: boolean;
}

export async function listApps(): Promise<AppRegistration[]> {
  const res = await fetch("/api/apps");
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error ?? `Status ${res.status}`);
  }
  return res.json();
}

export async function listPublicApps(): Promise<DirectoryApp[]> {
  const res = await fetch("/api/apps/directory");
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error ?? `Status ${res.status}`);
  }
  return res.json();
}

export async function createApp(req: CreateAppRequest): Promise<AppRegistration> {
  const res = await fetch("/api/apps", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error ?? `Status ${res.status}`);
  }
  return res.json();
}

export async function getApp(id: string): Promise<AppRegistration> {
  const res = await fetch(`/api/apps/${encodeURIComponent(id)}`);
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error ?? `Status ${res.status}`);
  }
  return res.json();
}

export async function updateApp(id: string, req: Partial<CreateAppRequest>): Promise<AppRegistration> {
  const res = await fetch(`/api/apps/${encodeURIComponent(id)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error ?? `Status ${res.status}`);
  }
  return res.json();
}

export async function deleteApp(id: string): Promise<void> {
  const res = await fetch(`/api/apps/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error ?? `Status ${res.status}`);
  }
}

export async function rotateSecret(id: string): Promise<{ client_secret: string }> {
  const res = await fetch(`/api/apps/${encodeURIComponent(id)}/secret`, {
    method: "POST",
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error ?? `Status ${res.status}`);
  }
  return res.json();
}

export async function uploadAppLogo(id: string, file: File): Promise<void> {
  const form = new FormData();
  form.append("file", file);
  const res = await fetch(`/api/apps/${encodeURIComponent(id)}/logo`, {
    method: "PUT",
    body: form,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error ?? `Status ${res.status}`);
  }
}

// ---- Admin API ----

export async function adminListApps(status?: string): Promise<AppRegistration[]> {
  const url = status ? `/api/admin/apps?status=${encodeURIComponent(status)}` : "/api/admin/apps";
  const res = await fetch(url);
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error ?? `Status ${res.status}`);
  }
  return res.json();
}

export async function adminApproveApp(id: string): Promise<AppRegistration> {
  const res = await fetch(`/api/admin/apps/${encodeURIComponent(id)}/approve`, {
    method: "POST",
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error ?? `Status ${res.status}`);
  }
  return res.json();
}

export async function adminRejectApp(id: string, reason?: string): Promise<AppRegistration> {
  const res = await fetch(`/api/admin/apps/${encodeURIComponent(id)}/reject`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ reason: reason ?? "" }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error ?? `Status ${res.status}`);
  }
  return res.json();
}

