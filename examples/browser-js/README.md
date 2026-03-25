# SCID Browser JS Example (PKCE)

A minimal vanilla-JS single-page app demonstrating OIDC authentication with SCID using the **PKCE flow** (no client secret required — safe for browser-only apps).

## What it shows

- Generating a PKCE code verifier / challenge
- Redirecting the user to SCID's authorization endpoint
- Exchanging the authorization code for tokens
- Fetching user info and displaying it

## Setup

1. **Register your app on SCID**
   - Go to `https://<your-scid>/apps` and register a new application.
   - Set **Client Type** to `Public` and enable **PKCE required**.
   - Add `http://localhost:8080/callback.html` as a redirect URI.
   - Note your **Client ID**.

2. **Edit `app.js`** — set the three constants at the top:

   ```js
   const SCID_BASE   = 'https://<your-scid>';   // no trailing slash
   const CLIENT_ID   = '<your-client-id>';
   const REDIRECT_URI = 'http://localhost:8080/callback.html';
   ```

3. **Serve the files** over HTTP (browsers block `file://` OAuth redirects):

   ```bash
   # Python 3
   python3 -m http.server 8080
   # or Node.js
   npx serve -p 8080 .
   ```

4. Open `http://localhost:8080` in your browser and click **Sign in with SCID**.

## Files

| File | Purpose |
|------|---------|
| `index.html` | Landing page with login button |
| `callback.html` | Handles the OIDC redirect and token exchange |
| `app.js` | All PKCE + OIDC logic (shared by both pages) |

## Source

Part of the [SCID project](https://github.com/example/scid/).
