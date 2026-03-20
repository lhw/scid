export interface VerifyStatus {
  authenticated: boolean;
  verified: boolean;
  handle: string;
  verified_at: string;
  pending_handle: string;
  pending_expires_at: string;
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

import { getAccessToken } from "./auth.js";

function authHeader(): Record<string, string> {
  const token = getAccessToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}

// Request a one-use Pocket ID signup token so the frontend can redirect a new
// user to Pocket ID's /signup page.
export async function getSignupToken(): Promise<string> {
  const res = await fetch("/api/auth/signup-token", { method: "POST" });
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
  const res = await fetch("/api/verify/status", { headers: authHeader() });
  if (!res.ok) throw new Error(`Status ${res.status}`);
  return res.json();
}

// Start a new verification for the given RSI handle.
export async function startVerify(
  handle: string,
): Promise<StartVerifyResponse> {
  const res = await fetch("/api/verify/start", {
    method: "POST",
    headers: { "Content-Type": "application/json", ...authHeader() },
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
    headers: authHeader(),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error ?? `Status ${res.status}`);
  }
  return res.json();
}
