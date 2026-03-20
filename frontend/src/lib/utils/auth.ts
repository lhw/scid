/**
 * OIDC PKCE helpers for authenticating against Pocket ID.
 *
 * Flow:
 *  1. login(returnPath)     — generate PKCE params, redirect to Pocket ID /authorize
 *  2. handleCallback(...)   — exchange code for access token, store in sessionStorage
 *  3. getAccessToken()      — return stored token for use in fetch calls
 */

// These are baked in at build time by SvelteKit / Vite.
// Set PUBLIC_POCKET_ID_URL and PUBLIC_OIDC_CLIENT_ID in your .env file.
import * as staticPublic from '$env/static/public';
const _env = staticPublic as unknown as Record<string, string | undefined>;

const POCKET_ID_URL = _env.PUBLIC_POCKET_ID_URL ?? 'https://id.scid.my';
const CLIENT_ID = _env.PUBLIC_OIDC_CLIENT_ID ?? 'scid-frontend';

const TOKEN_KEY = 'scid_access_token';
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

// ── Token storage ─────────────────────────────────────────────────────────────

export function getAccessToken(): string | null {
  if (typeof sessionStorage === 'undefined') return null;
  return sessionStorage.getItem(TOKEN_KEY);
}

export function setAccessToken(token: string): void {
  sessionStorage.setItem(TOKEN_KEY, token);
}

export function clearAccessToken(): void {
  sessionStorage.removeItem(TOKEN_KEY);
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
    client_id: CLIENT_ID,
    redirect_uri: redirectUri,
    scope: 'openid profile email',
    code_challenge: challenge,
    code_challenge_method: 'S256',
    state,
  });

  window.location.href = `${POCKET_ID_URL}/authorize?${params}`;
}

/**
 * Complete the PKCE flow after Pocket ID redirects back to /callback.
 * Validates state, exchanges code for tokens, stores the access token.
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

  const redirectUri = `${window.location.origin}/callback`;
  const res = await fetch(`${POCKET_ID_URL}/api/oidc/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({
      grant_type: 'authorization_code',
      client_id: CLIENT_ID,
      redirect_uri: redirectUri,
      code,
      code_verifier: verifier,
    }),
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({})) as Record<string, string>;
    throw new Error(body.error_description ?? body.error ?? `Token exchange failed (${res.status})`);
  }

  const data = await res.json() as { access_token: string };
  setAccessToken(data.access_token);

  try {
    const parsed = JSON.parse(atob(state)) as { returnPath?: string };
    return parsed.returnPath ?? '/';
  } catch {
    return '/';
  }
}
