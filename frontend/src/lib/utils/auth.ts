/**
 * OIDC PKCE helpers for authenticating against Pocket ID.
 *
 * Flow:
 *  1. login(returnPath)     — generate PKCE params, redirect to Pocket ID /authorize
 *  2. handleCallback(...)   — exchange code via the companion backend, which sets an httpOnly session cookie
 *  3. logout()              — clears the backend session cookie
 */

import { PUBLIC_OIDC_CLIENT_ID, PUBLIC_POCKET_ID_URL } from '$lib/utils/public-env';

const VERIFIER_KEY = 'scid_pkce_verifier';
const STATE_KEY = 'scid_oauth_state';

// ── PKCE helpers ──────────────────────────────────────────────────────────────

function base64url(buf: ArrayBuffer | Uint8Array): string {
  const bytes = buf instanceof Uint8Array ? buf : new Uint8Array(buf);
  let str = '';
  for (const b of bytes) str += String.fromCharCode(b);
  return btoa(str).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
}

function generateVerifier(): string {
  const arr = new Uint8Array(32);
  crypto.getRandomValues(arr);
  return base64url(arr);
}

async function generateChallenge(verifier: string): Promise<string> {
  const data = new TextEncoder().encode(verifier);
  const hash = await crypto.subtle.digest('SHA-256', data);
  return base64url(hash);
}

// ── Auth flow ─────────────────────────────────────────────────────────────────

/**
 * Initiate a PKCE authorization code flow. Redirects the browser to Pocket ID.
 * @param returnPath  URL path to return to after successful login (default: '/verify')
 */
export async function login(returnPath = '/verify'): Promise<void> {
  const verifier = generateVerifier();
  const challenge = await generateChallenge(verifier);
  // Store a random nonce inside state to prevent CSRF
  const state = btoa(JSON.stringify({ returnPath, r: crypto.randomUUID() }));

  sessionStorage.setItem(VERIFIER_KEY, verifier);
  sessionStorage.setItem(STATE_KEY, state);

  const redirectUri = `${window.location.origin}/callback`;
  const params = new URLSearchParams({
    response_type: 'code',
    client_id: PUBLIC_OIDC_CLIENT_ID,
    redirect_uri: redirectUri,
    scope: 'openid profile email',
    code_challenge: challenge,
    code_challenge_method: 'S256',
    state,
  });

  window.location.href = `${PUBLIC_POCKET_ID_URL}/authorize?${params}`;
}

/**
 * Complete the PKCE flow after Pocket ID redirects back to /callback.
 * Validates state and asks the companion backend to exchange the code and set
 * an httpOnly session cookie.
 * @returns the returnPath that was passed to login()
 */
export async function handleCallback(code: string, state: string): Promise<string> {
  const storedState = sessionStorage.getItem(STATE_KEY);
  const verifier = sessionStorage.getItem(VERIFIER_KEY);

  if (!storedState || storedState !== state) {
    throw new Error('Invalid OAuth state — possible CSRF attempt');
  }
  if (!verifier) {
    throw new Error('Missing PKCE verifier — please try logging in again');
  }

  sessionStorage.removeItem(STATE_KEY);
  sessionStorage.removeItem(VERIFIER_KEY);

  const res = await fetch('/api/auth/callback', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      code,
      code_verifier: verifier,
    }),
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({})) as Record<string, string>;
    throw new Error(body.error_description ?? body.error ?? `Token exchange failed (${res.status})`);
  }

  try {
    const parsed = JSON.parse(atob(state)) as { returnPath?: string };
    return parsed.returnPath ?? '/';
  } catch {
    return '/';
  }
}

export async function logout(): Promise<void> {
  await fetch('/api/auth/logout', { method: 'POST' });
}
