/**
 * SCID PKCE OIDC helper — shared by index.html and callback.html.
 *
 * Edit the three constants below to match your SCID registration.
 */

const SCID_BASE    = 'https://<your-scid>';          // no trailing slash
const CLIENT_ID    = '<your-client-id>';
const REDIRECT_URI = 'http://localhost:8080/callback.html';

// ── PKCE helpers ─────────────────────────────────────────────────────────────

function randomBytes(length) {
  return crypto.getRandomValues(new Uint8Array(length));
}

function base64url(bytes) {
  return btoa(String.fromCharCode(...bytes))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=/g, '');
}

async function sha256(plain) {
  const encoded = new TextEncoder().encode(plain);
  const digest  = await crypto.subtle.digest('SHA-256', encoded);
  return new Uint8Array(digest);
}

async function generatePKCE() {
  const verifier  = base64url(randomBytes(32));
  const challenge = base64url(await sha256(verifier));
  return { verifier, challenge };
}

// ── Auth flow ─────────────────────────────────────────────────────────────────

/**
 * Start the PKCE login flow: generate verifier/state, save them in
 * sessionStorage, then redirect to the SCID authorization endpoint.
 */
async function startLogin() {
  const { verifier, challenge } = await generatePKCE();
  const state = base64url(randomBytes(16));

  sessionStorage.setItem('pkce_verifier', verifier);
  sessionStorage.setItem('oauth_state',   state);

  const params = new URLSearchParams({
    response_type:         'code',
    client_id:             CLIENT_ID,
    redirect_uri:          REDIRECT_URI,
    scope:                 'openid profile email',
    code_challenge:        challenge,
    code_challenge_method: 'S256',
    state,
  });

  window.location.href = `${SCID_BASE}/authorize?${params}`;
}

/**
 * Complete the PKCE flow on the callback page.
 * Returns the token response object, or throws on error.
 */
async function handleCallback() {
  const params   = new URLSearchParams(window.location.search);
  const code     = params.get('code');
  const state    = params.get('state');
  const errorMsg = params.get('error_description') || params.get('error');

  if (errorMsg) throw new Error(`Authorization failed: ${errorMsg}`);
  if (!code)    throw new Error('No authorization code in callback URL.');

  const savedState = sessionStorage.getItem('oauth_state');
  if (!savedState || savedState !== state) {
    throw new Error('State mismatch — possible CSRF attack. Please try again.');
  }

  const verifier = sessionStorage.getItem('pkce_verifier');
  if (!verifier) throw new Error('No PKCE verifier found. Please restart the login flow.');

  sessionStorage.removeItem('pkce_verifier');
  sessionStorage.removeItem('oauth_state');

  const body = new URLSearchParams({
    grant_type:    'authorization_code',
    client_id:     CLIENT_ID,
    redirect_uri:  REDIRECT_URI,
    code,
    code_verifier: verifier,
  });

  const res = await fetch(`${SCID_BASE}/api/oidc/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body,
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.error_description || err.error || `Token exchange failed (${res.status})`);
  }

  return res.json();
}

/**
 * Fetch the OIDC userinfo endpoint using an access token.
 */
async function fetchUserInfo(accessToken) {
  const res = await fetch(`${SCID_BASE}/api/oidc/userinfo`, {
    headers: { Authorization: `Bearer ${accessToken}` },
  });
  if (!res.ok) throw new Error(`Userinfo fetch failed (${res.status})`);
  return res.json();
}
